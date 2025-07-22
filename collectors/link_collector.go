package collectors

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// CollectLinks discovers and returns all links from a starting URL using the provided config
func CollectLinks(startURL string, config *CollectorConfig, sameHostOnly bool) ([]LinkInfo, string, error) {
	// Parse the starting URL to get the host
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}
	hostname := parsedURL.Hostname()

	var links []LinkInfo
	bookTitle := "Default Title"
	linkOrder := 0

	// Create a collector just for discovering links
	linkCollector := colly.NewCollector(
		colly.MaxDepth(0),
	)

	// Set up the callback for link discovery
	linkCollector.OnHTML("html", func(e *colly.HTMLElement) {
		// Extract book title
		title := e.DOM.Find(config.TitleSelector).Text()
		if title != "" {
			bookTitle = title
		}

		// Find and collect all links on the page using the configured selector
		e.DOM.Find(config.LinkSelector).Each(func(i int, s *goquery.Selection) {
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
			if sameHostOnly && linkURL.Hostname() != hostname {
				return
			}

			// Skip configured file extensions
			ext := strings.ToLower(path.Ext(linkURL.Path))
			for _, skipExt := range config.SkipExtensions {
				if ext == skipExt {
					return
				}
			}

			isSubSection := false
			oldLink := link
			if len(config.LinkReplace) > 0 {
				for _, replacement := range config.LinkReplace {
					link = strings.ReplaceAll(link, replacement.OldText, replacement.NewText)
					if oldLink != link && replacement.IsSubSection {
						isSubSection = true
					}
				}
			}

			if !strings.Contains(link, config.LinkFilter) {
				// Store the link with its order
				// link = strings.ReplaceAll(link, "/_contents", "")
				links = append(links, LinkInfo{
					URL:          link,
					Order:        linkOrder,
					IsSubSection: isSubSection,
				})
				linkOrder++
			}
		})
	})

	// Start link discovery
	fmt.Printf("Discovering links at %s\n", startURL)
	err = linkCollector.Visit(startURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to visit starting URL: %w", err)
	}

	linkCollector.Wait()

	// Truncate to 10 links for testing
	var testLinksList []LinkInfo
	for _, link := range links {
		if link.Order < 10 {
			fmt.Printf("%s\n", link.URL)
			testLinksList = append(testLinksList, link)
		}
	}

	links = testLinksList

	return links, bookTitle, nil
}
