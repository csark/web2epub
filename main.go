package main

//TODO: Look at different logging packages - logrus, zap, zerolog
//TODO: Add footnotes - https://ebooks.stackexchange.com/questions/109/how-can-i-put-footnotes-in-an-ebook, will have to extract the footnotes and add them as a separate section using this class name class="panelGridLayout-ZuE9Q"

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-epub"
	"github.com/gocolly/colly/v2"
)

// PageContent stores the extracted content from a web page
type PageContent struct {
	URL          string
	Title        string
	Author       string
	Content      *goquery.Selection
	Order        int
	isSubSection bool
}

func main() {
	// Define command line flags
	startURL := flag.String("url", "", "Starting URL to crawl (required)")
	outputFile := flag.String("output", "", "Will grab title from title of first page unless this flag is specified")
	coverImg := flag.String("cover", "", "URL of desired cover image. Defaults to no cover image")
	// Does not support user definable maxDepth at this time
	//maxDepth := flag.Int("depth", 1, "Maximum crawl depth")
	sameHostOnly := flag.Bool("same-host", true, "Only crawl pages on the same host")
	flag.Parse()

	// Validate required flags
	if *startURL == "" {
		fmt.Println("Error: Starting URL is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Parse the starting URL to get the host
	parsedURL, err := url.Parse(*startURL)
	if err != nil {
		log.Fatal("Invalid URL:", err)
	}
	hostname := parsedURL.Hostname()

	// Store the collected pages
	pages := make(map[string]*PageContent)

	// Create a temporary directory for resource files
	tempDir, err := os.MkdirTemp("", "epub-builder")
	if err != nil {
		log.Fatal("Failed to create temp directory:", err)
	}
	defer os.RemoveAll(tempDir)

	// First, collect all links in order
	type LinkInfo struct {
		URL   string
		Order int
	}
	bookTitle := "Default Title"
	var links []LinkInfo
	linkOrder := 0

	// Create a collector just for discovering links
	linkCollector := colly.NewCollector(
		colly.MaxDepth(0),
	)

	// Set up the callback for link discovery
	linkCollector.OnHTML("html", func(e *colly.HTMLElement) {
		bookTitle = e.DOM.Find("title").Text()
		if bookTitle == "" {
			bookTitle = *outputFile
		}

		// Find and collect all links on the page
		e.DOM.Find("a[href].list-tile").Each(func(i int, s *goquery.Selection) {
			link, exists := s.Attr("href")
			if !exists {
				return
			}

			// Skip links to other domains if sameHostOnly is true
			linkURL, err := url.Parse(link)
			if err != nil {
				return
			}

			// If the link is relative, make it absolute
			if !linkURL.IsAbs() {
				linkURL = e.Request.URL.ResolveReference(linkURL)
				link = linkURL.String()
			}

			// Skip external links if sameHostOnly is true
			if *sameHostOnly && linkURL.Hostname() != hostname {
				return
			}

			// Skip common file types that aren't HTML
			ext := strings.ToLower(path.Ext(linkURL.Path))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" ||
				ext == ".pdf" || ext == ".zip" || ext == ".mp3" || ext == ".mp4" {
				return
			}

			// Store the link with its order
			links = append(links, LinkInfo{
				URL:   link,
				Order: linkOrder,
			})
			linkOrder++
		})
	})

	// Start link discovery
	fmt.Printf("Discovering links at %s\n", *startURL)
	linkCollector.Visit(*startURL)
	linkCollector.Wait()

	// Now process the pages in parallel while preserving order
	pageCollector := colly.NewCollector(
		colly.MaxDepth(0), // We already have all links, no need to crawl further
		colly.Async(true),
	)

	// Set up parallel processing
	pageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 6,
		Delay:       3 * time.Second,
	})

	// Set up the callback for page processing
	pageCollector.OnHTML("html", func(e *colly.HTMLElement) {
		pageURL := e.Request.URL.String()

		// Find the order of this page in our links list
		var pageOrder int
		for _, link := range links {
			if link.URL == pageURL {
				pageOrder = link.Order
				break
			}
		}

		// Extract the title
		title := e.DOM.Find("title").Text()
		if title == "" {
			title = fmt.Sprintf("Page %d", pageOrder+1)
		}

		author := e.DOM.Find(".author-name").Text()
		if author == "" {
			author = "General Authority"
		} else {
			replacer := strings.NewReplacer(
				"By", "",
				"President", "",
				"Elder", "",
				"Sister", "",
				"Brother", "",
			)
			author = replacer.Replace(author)
		}

		// Extract the main content
		article := e.DOM.Find("article")
		if article.Length() > 0 {
			// Remove unwanted elements from article content
			article.Find("script, footer, iframe, button, .nav, .menu, .sidebar, .ad, .ads, .fbfDMV").Remove()
		} else {
			log.Print("'article' element not found, falling back to generic content discovery...")
			// Fallback to body content with some cleaning
			e.DOM.Find("body").Each(func(i int, s *goquery.Selection) {
				// Remove scripts, styles, nav, etc. .fbfDMV - timestamp on header image
				s.Find("script, style, nav, header, footer, iframe, button, .nav, .menu, .sidebar, .ad, .ads, .fbfDMV").Remove()
				article = s
			})
		}

		isSubSection := true
		// Check if page is a section heading
		contentLength := len(article.Text())
		// log.Printf("Content length is: %d", contentLength)
		if contentLength < 100 {
			isSubSection = false
		}

		// Download images in the goroutine
		e.DOM.Find("img").Each(func(i int, s *goquery.Selection) {
			imgURL, exists := s.Attr("src")
			if exists {
				output_path, err := downloadImage(imgURL, tempDir)
				if err != nil {
					log.Printf("Error downloading image %s: %v", imgURL, err)
				}
				// Create a new img tag with just the src attribute
				newImg := fmt.Sprintf(`<img src="%s">`, output_path)
				s.ReplaceWithHtml(newImg)
			}
		})

		// Store the page content
		pages[pageURL] = &PageContent{
			URL:          pageURL,
			Title:        title,
			Author:       author,
			Content:      article,
			Order:        pageOrder,
			isSubSection: isSubSection,
		}
	})

	// Set up error handling
	pageCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error visiting %s: %v", r.Request.URL, err)
	})

	// Process all discovered links
	fmt.Printf("Processing %d discovered pages\n", len(links))
	for _, link := range links {
		pageCollector.Visit(link.URL)
	}

	// Wait for all pages to be processed
	pageCollector.Wait()

	css := `h1 {
    margin-block-end: 0.33em;
}
h3 {
    margin-block-start: 0;
    margin-block-end: 0;
}
p.author-name, p.author-role {
    margin-block-start: 0;
    margin-block-end: 0;
}
p {
    margin-block-start: 0;
    margin-block-end: 0.5em;
}
p.kicker {
    font-style: italic;
	margin-block-start: 1em;
    margin-block-end: 2em;
}`

	// Write CSS to a file in the temp directory
	cssPath := path.Join(tempDir, "styles.css")
	err = os.WriteFile(cssPath, []byte(css), 0644)
	if err != nil {
		log.Fatal("Error writing CSS file:", err)
	}

	// Create the EPUB book
	book, err := epub.NewEpub(bookTitle)
	if err != nil {
		log.Fatal("Error creating EPUB:", err)
	}
	book.SetTitle(bookTitle)
	book.SetAuthor("Church of Jesus Christ of Latter-day Saints")
	book.SetDescription(fmt.Sprintf("Content crawled from %s on %s by casrk/web2epub", hostname, time.Now().Format("2006-01-02")))
	cssPath, err = book.AddCSS(cssPath, "")
	if err != nil {
		log.Fatal("Error adding CSS:", err)
	}

	// Sort pages by order
	sortedPages := make([]*PageContent, len(pages))
	for _, page := range pages {
		sortedPages[page.Order] = page
	}

	SectionLink := ""

	// Add each page to the EPUB
	for _, page := range sortedPages {
		// Create a section in the EPUB
		var contentBuilder strings.Builder

		// Find all of the img tags in the article
		page.Content.Find("img").Each(func(i int, s *goquery.Selection) {
			tmp_path, exists := s.Attr("src")
			if exists {
				ebook_path, err := book.AddImage(tmp_path, "")
				if err != nil {
					log.Fatal("Error processing image:", err)
				}
				// Create a new img tag with just the src attribute
				newImg := fmt.Sprintf(`<img src="%s">`, ebook_path)
				s.ReplaceWithHtml(newImg)
			}
		})

		// Edit author attributes
		authorName := page.Content.Find(".author-name")
		authorName.SetHtml(fmt.Sprintf("<h3>%s</h3>", authorName.Text()))

		authorRole := page.Content.Find(".author-role")
		authorRole.SetHtml(fmt.Sprintf("<h3>%s</h3>", authorRole.Text()))

		content, _ := page.Content.First().Html()
		contentBuilder.WriteString(content)

		title := fmt.Sprintf("%s - %s", page.Title, page.Author)

		if page.isSubSection {
			// log.Printf("Subsection")
			_, err := book.AddSubSection(SectionLink, contentBuilder.String(), title, "", cssPath)
			if err != nil {
				log.Printf("Error adding subsection for %s: %v", page.URL, err)
			}
		} else {
			// log.Printf("Section")
			relativePath, err := book.AddSection(contentBuilder.String(), title, "", cssPath)
			if err != nil {
				log.Printf("Error adding section for %s: %v", page.URL, err)
			}
			SectionLink = relativePath
		}

		fmt.Printf("Added page: %s\n", title)
	}

	//Add cover image
	if *coverImg != "" {
		output_path, err := downloadImage(*coverImg, tempDir)
		if err != nil {
			log.Printf("Error downloading cover image %s: %v", *coverImg, err)
		}
		ebook_path, err := book.AddImage(output_path, "")
		if err != nil {
			log.Fatal("Error processing cover image:", err)
		}
		book.SetCover(ebook_path, "")
	}

	// Save the EPUB file
	err = book.Write(bookTitle + ".epub")
	if err != nil {
		log.Fatal("Error writing EPUB:", err)
	}

	fmt.Printf("\nSuccessfully created EPUB: %s\n", bookTitle)
	fmt.Printf("Total pages: %d\n", len(pages))
}

// downloadImage downloads an image from a URL to the specified directory
func downloadImage(imageURL, dir string) (string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Create a file with a random name
	// Generate 16 random bytes
	b := make([]byte, 16)
	rand.Read(b)
	// Convert to hex string and add .jpg extension
	filename := fmt.Sprintf("%x", b)

	// log.Printf("Filename: %s", filename)

	filepath := path.Join(dir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Copy the image data to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}
