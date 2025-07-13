package collectors

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// CollectPages processes the discovered links and extracts content from each page
func CollectPages(links []LinkInfo, config *CollectorConfig, tempDir string, downloadImageFunc func(string, string) (string, error)) (map[string]*PageContent, error) {
	pages := make(map[string]*PageContent)

	// Create collector for page processing
	pageCollector := colly.NewCollector(
		colly.MaxDepth(0), // We already have all links, no need to crawl further
		colly.Async(true),
	)

	// Set up parallel processing with config
	pageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: config.Parallelism,
		Delay:       time.Duration(config.DelaySeconds) * time.Second,
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
		title := e.DOM.Find(config.TitleSelector).Text()
		if title == "" {
			title = fmt.Sprintf("Page %d", pageOrder+1)
		}

		// Extract and process author
		author := e.DOM.Find(config.AuthorSelector).Text()
		if author == "" {
			author = config.DefaultAuthor
		} else {
			// Apply configured author replacements
			for old, new := range config.AuthorReplacements {
				author = strings.ReplaceAll(author, old, new)
			}
			author = strings.TrimSpace(author)
		}

		// Extract the main content using configured selector
		var article *goquery.Selection
		content := e.DOM.Find(config.ContentSelector)
		if content.Length() > 0 {
			article = content
			// Remove unwanted elements from article content
			for _, selector := range config.RemoveSelectors {
				article.Find(selector).Remove()
			}
		} else if config.FallbackToBody {
			log.Printf("'%s' element not found for %s, falling back to body content...", config.ContentSelector, pageURL)
			// Fallback to body content with cleaning
			e.DOM.Find("body").Each(func(i int, s *goquery.Selection) {
				// Remove unwanted elements
				for _, selector := range config.RemoveSelectors {
					s.Find(selector).Remove()
				}
				article = s
			})
		} else {
			log.Printf("'%s' element not found for %s and fallback disabled", config.ContentSelector, pageURL)
			return
		}

		// Determine if this is a subsection based on content length
		isSubSection := true
		contentLength := len(article.Text())
		if contentLength < config.SubSectionThreshold {
			isSubSection = false
		}

		// Download images if downloadImageFunc is provided
		if downloadImageFunc != nil {
			e.DOM.Find("img").Each(func(i int, s *goquery.Selection) {
				imgURL, exists := s.Attr("src")
				if exists {
					outputPath, err := downloadImageFunc(imgURL, tempDir)
					if err != nil {
						log.Printf("Error downloading image %s: %v", imgURL, err)
					} else {
						// Create a new img tag with just the src attribute
						newImg := fmt.Sprintf(`<img src="%s">`, outputPath)
						s.ReplaceWithHtml(newImg)
					}
				}
			})
		}

		// Store the page content
		pages[pageURL] = &PageContent{
			URL:          pageURL,
			Title:        title,
			Author:       author,
			Content:      article,
			Order:        pageOrder,
			IsSubSection: isSubSection,
		}
	})

	// Set up error handling
	pageCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error visiting %s: %v", r.Request.URL, err)
	})

	// Process all discovered links
	fmt.Printf("Processing %d discovered pages\n", len(links))
	for _, link := range links {
		err := pageCollector.Visit(link.URL)
		if err != nil {
			log.Printf("Error queuing %s: %v", link.URL, err)
		}
	}

	// Wait for all pages to be processed
	pageCollector.Wait()

	return pages, nil
}