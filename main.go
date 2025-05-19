package main

import (
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
	"github.com/bmaupin/go-epub"
	"github.com/gocolly/colly/v2"
)

// PageContent stores the extracted content from a web page
type PageContent struct {
	URL      string
	Title    string
	Content  string
	Order    int
	FilePath string
}

func main() {
	// Define command line flags
	startURL := flag.String("url", "", "Starting URL to crawl (required)")
	outputFile := flag.String("output", "output.epub", "Output EPUB file name")
	maxDepth := flag.Int("depth", 1, "Maximum crawl depth")
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

	// Initialize the collector
	c := colly.NewCollector(
		colly.MaxDepth(*maxDepth),
		colly.Async(true),
	)

	// Set up parallel processing
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 4,
		Delay:       1 * time.Second,
	})

	// Store the collected pages
	pages := make(map[string]*PageContent)
	pageOrder := 0

	// Create a temporary directory for HTML files
	tempDir, err := os.MkdirTemp("", "epub-builder")
	if err != nil {
		log.Fatal("Failed to create temp directory:", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up the callback for when a page is visited
	c.OnHTML("html", func(e *colly.HTMLElement) {
		// Skip if not on the same host and sameHostOnly is true
		pageURL := e.Request.URL.String()
		currentHostname := e.Request.URL.Hostname()
		
		if *sameHostOnly && currentHostname != hostname {
			return
		}

		// Skip if already processed
		if _, exists := pages[pageURL]; exists {
			return
		}

		// Extract the title
		title := e.DOM.Find("title").Text()
		if title == "" {
			title = fmt.Sprintf("Page %d", pageOrder+1)
		}

		// Extract the main content
		// This is a simplistic approach - you might want to use something more sophisticated
		var contentBuilder strings.Builder
		
		// Add article/main content if it exists
		article := e.DOM.Find("article, main, .content, #content")
		if article.Length() > 0 {
			content, _ := article.First().Html()
			contentBuilder.WriteString(content)
		} else {
			// Fallback to body content with some cleaning
			e.DOM.Find("body").Each(func(i int, s *goquery.Selection) {
				// Remove scripts, styles, nav, etc.
				s.Find("script, style, nav, header, footer, .nav, .menu, .sidebar, .ad, .ads").Remove()
				content, _ := s.Html()
				contentBuilder.WriteString(content)
			})
		}

		// Create an HTML file for this page
		fileName := fmt.Sprintf("page_%d.html", pageOrder)
		filePath := path.Join(tempDir, fileName)
		
		htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <meta charset="utf-8">
</head>
<body>
    %s
</body>
</html>`, title, contentBuilder.String())
		
		err := os.WriteFile(filePath, []byte(htmlContent), 0644)
		if err != nil {
			log.Printf("Error writing HTML file for %s: %v", pageURL, err)
			return
		}

		// Store the page content
		pages[pageURL] = &PageContent{
			URL:      pageURL,
			Title:    title,
			Content:  contentBuilder.String(),
			Order:    pageOrder,
			FilePath: filePath,
		}
		pageOrder++

		// Find and visit all links on the page
		e.DOM.Find("a[href]").Each(func(i int, s *goquery.Selection) {
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

			e.Request.Visit(link)
		})
	})

	// Set up error handling
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error visiting %s: %v", r.Request.URL, err)
	})

	// Start crawling
	fmt.Printf("Starting crawl at %s (max depth: %d)\n", *startURL, *maxDepth)
	c.Visit(*startURL)

	// Wait for all requests to finish
	c.Wait()

	// Create the EPUB book
	book := epub.NewEpub(fmt.Sprintf("Content from %s", hostname))
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
		_, err := book.AddSection(page.Content, page.Title, "", "")
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

	// Create a file with the image name
	parsedURL, _ := url.Parse(imageURL)
	filename := path.Base(parsedURL.Path)
	if filename == "" || filename == "." {
		filename = fmt.Sprintf("image_%d.jpg", time.Now().UnixNano())
	}

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
