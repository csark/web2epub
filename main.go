package main

//TODO: Look at different logging packages - logrus, zap, zerolog
//TODO: Look at pulling a better header image from the available options "srcset"
//TODO: Generate better ebook name
//TODO: Pull images during crawl, then use updated src path for the ebook insertion
//TODO: Header and kicker text formatting in ebook should be updated
//TODO: Fix article order

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
	URL     string
	Title   string
	Content *goquery.Selection
	Order   int
}

func main() {
	// Define command line flags
	startURL := flag.String("url", "", "Starting URL to crawl (required)")
	outputFile := flag.String("output", "output.epub", "Output EPUB file name")
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
	var links []LinkInfo
	linkOrder := 0

	// Create a collector just for discovering links
	linkCollector := colly.NewCollector(
		colly.MaxDepth(0),
	)

	// Set up the callback for link discovery
	linkCollector.OnHTML("html", func(e *colly.HTMLElement) {
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
		Delay:       5 * time.Second,
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

		// Store the page content
		pages[pageURL] = &PageContent{
			URL:     pageURL,
			Title:   title,
			Content: article,
			Order:   pageOrder,
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

	// Create the EPUB book
	book, err := epub.NewEpub(fmt.Sprintf("Content from %s", hostname))
	if err != nil {
		log.Fatal("Error creating EPUB:", err)
	}
	book.SetAuthor("Web Crawler")
	book.SetDescription(fmt.Sprintf("Content crawled from %s on %s", hostname, time.Now().Format("2006-01-02")))

	// Sort pages by order
	sortedPages := make([]*PageContent, len(pages))
	for _, page := range pages {
		sortedPages[page.Order] = page
	}

	// Add each page to the EPUB
	for _, page := range sortedPages {
		// Create a section in the EPUB
		var contentBuilder strings.Builder

		// Find all of the img tags in the article
		page.Content.Find("img").Each(func(i int, s *goquery.Selection) {
			imgURL, exists := s.Attr("src")
			if exists {
				output_path, err := downloadImage(imgURL, tempDir)
				if err != nil {
					log.Printf("Error downloading image %s: %v", imgURL, err)
				}
				ebook_path, err := book.AddImage(output_path, "")
				if err != nil {
					log.Fatal("Error processing image:", err)
				}

				// Create a new img tag with just the src attribute
				newImg := fmt.Sprintf(`<img src="%s">`, ebook_path)
				s.ReplaceWithHtml(newImg)
			}
		})
		content, _ := page.Content.First().Html()
		contentBuilder.WriteString(content)

		_, err := book.AddSection(contentBuilder.String(), page.Title, "", "")
		if err != nil {
			log.Printf("Error adding section for %s: %v", page.URL, err)
		}

		fmt.Printf("Added page: %s\n", page.Title)
	}

	// Save the EPUB file
	err = book.Write(*outputFile)
	if err != nil {
		log.Fatal("Error writing EPUB:", err)
	}

	fmt.Printf("\nSuccessfully created EPUB: %s\n", *outputFile)
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
