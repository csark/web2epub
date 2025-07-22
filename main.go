package main

//TODO: Look at different logging packages - logrus, zap, zerolog

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"web2epub/collectors"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-epub"
)

func main() {
	// Define command line flags
	startURL := flag.String("url", "", "Starting URL to crawl (required)")
	outputFile := flag.String("output", "", "Will grab title from title of first page unless this flag is specified")
	coverImg := flag.String("cover", "", "URL of desired cover image. Defaults to no cover image")
	module := flag.String("module", "conference", "Collection module to use (conference, scriptures, ensign)")
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

	// Get the collector configuration for the specified module
	config, err := collectors.GetConfigByModule(*module)
	if err != nil {
		log.Fatal("Module configuration error:", err)
	}

	// Store the collected pages
	pages := make(map[string]*collectors.PageContent)

	// Create a temporary directory for resource files
	tempDir, err := os.MkdirTemp("", "epub-builder")
	if err != nil {
		log.Fatal("Failed to create temp directory:", err)
	}
	defer os.RemoveAll(tempDir)

	// Use the modular collectors to discover links
	links, bookTitle, err := collectors.CollectLinks(*startURL, config, *sameHostOnly)
	if err != nil {
		log.Fatal("Link collection failed:", err)
	}

	// Use custom book title if provided
	if *outputFile != "" {
		bookTitle = *outputFile
	}

	// fmt.Printf("Book Title: %s\n", bookTitle)
	// fmt.Printf("Number of links found: %d\n", len(links))
	// for i, link := range links {
	// 	fmt.Printf(" %d. %s (Order: %d)\n", i+1, link.URL, link.Order)
	// }

	// os.Exit(0)

	// Use the modular collectors to process pages
	pages, err = collectors.CollectPages(links, config, tempDir, downloadImage)
	if err != nil {
		log.Fatal("Page collection failed:", err)
	}

	// Write CSS to a file in the temp directory
	cssPath := path.Join(tempDir, "styles.css")
	err = os.WriteFile(cssPath, []byte(config.CollectorCSS), 0644)
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
	book.SetDescription(fmt.Sprintf("Content crawled from %s on %s by casrk/web2epub", *startURL, time.Now().Format("2006-01-02")))
	cssPath, err = book.AddCSS(cssPath, "")
	if err != nil {
		log.Fatal("Error adding CSS:", err)
	}

	// Sort pages by order
	sortedPages := make([]*collectors.PageContent, len(pages))
	for _, page := range pages {
		// fmt.Printf("Current url: %s\n", page.URL)
		// fmt.Printf("Page number:%d\n", page.Order)
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

		if page.IsSubSection {
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
