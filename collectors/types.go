package collectors

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// PageContent stores the extracted content from a web page
type PageContent struct {
	URL          string
	Title        string
	Author       string
	Content      *goquery.Selection
	Order        int
	IsSubSection bool
}

// LinkInfo stores discovered link information
type LinkInfo struct {
	URL   string
	Order int
}

// CollectorConfig holds configuration for different collection strategies
type CollectorConfig struct {
	// Link discovery settings
	LinkSelector    string   // CSS selector for finding links
	LinkFilter      string   // String to match in a url
	TitleSelector   string   // CSS selector for page title
	AuthorSelector  string   // CSS selector for author
	ContentSelector string   // CSS selector for main content
	RemoveSelectors []string // CSS selectors for elements to remove

	// Author processing
	AuthorReplacements map[string]string // String replacements for author names
	DefaultAuthor      string            // Default author if none found

	// Content processing
	SubSectionThreshold int  // Content length threshold for subsections
	FallbackToBody      bool // Fall back to body if content selector fails

	// Crawling settings
	Parallelism    int      // Number of parallel requests
	DelaySeconds   int      // Delay between requests
	SkipExtensions []string // File extensions to skip
}

// GetGeneralConferenceConfig returns config for LDS General Conference pages
func GetGeneralConferenceConfig() *CollectorConfig {
	return &CollectorConfig{
		LinkSelector:    "a[href].list-tile",
		TitleSelector:   "title",
		AuthorSelector:  ".author-name",
		ContentSelector: "article",
		RemoveSelectors: []string{
			"script", "footer", "iframe", "button",
			".nav", ".menu", ".sidebar", ".ad", ".ads", ".fbfDMV",
		},
		AuthorReplacements: map[string]string{
			"By":        "",
			"President": "",
			"Elder":     "",
			"Sister":    "",
			"Brother":   "",
		},
		DefaultAuthor:       "General Authority",
		SubSectionThreshold: 100,
		FallbackToBody:      true,
		Parallelism:         6,
		DelaySeconds:        3,
		SkipExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif",
			".pdf", ".zip", ".mp3", ".mp4",
		},
	}
}

// GetScripturesConfig returns config for LDS Scripture pages
func GetScripturesConfig() *CollectorConfig {
	return &CollectorConfig{
		LinkSelector:    "a[href].list-tile",
		LinkFilter:      "illustrations",
		TitleSelector:   "title",
		AuthorSelector:  "",
		ContentSelector: ".body-block",
		RemoveSelectors: []string{
			"script", "footer", "iframe", "button",
			".nav", ".menu", ".sidebar", ".ad", ".ads",
			".study-notes", ".footnotes",
		},
		AuthorReplacements:  map[string]string{},
		DefaultAuthor:       "",
		SubSectionThreshold: 50,
		FallbackToBody:      true,
		Parallelism:         6,
		DelaySeconds:        2,
		SkipExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif",
			".pdf", ".zip", ".mp3", ".mp4",
		},
	}
}

// GetEnsignConfig returns config for LDS Ensign magazine articles
func GetEnsignConfig() *CollectorConfig {
	return &CollectorConfig{
		LinkSelector:    "a[href].title-link",
		TitleSelector:   "title",
		AuthorSelector:  ".author-name",
		ContentSelector: "article",
		RemoveSelectors: []string{
			"script", "footer", "iframe", "button",
			".nav", ".menu", ".sidebar", ".ad", ".ads",
			".kicker",
		},
		AuthorReplacements: map[string]string{
			"By": "",
		},
		DefaultAuthor:       "Church Author",
		SubSectionThreshold: 200,
		FallbackToBody:      true,
		Parallelism:         4,
		DelaySeconds:        3,
		SkipExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif",
			".pdf", ".zip", ".mp3", ".mp4",
		},
	}
}

// GetConfigByModule returns the appropriate collector config based on module name
func GetConfigByModule(module string) (*CollectorConfig, error) {
	switch strings.ToLower(module) {
	case "conference", "general-conference":
		return GetGeneralConferenceConfig(), nil
	case "scriptures":
		return GetScripturesConfig(), nil
	case "ensign":
		return GetEnsignConfig(), nil
	default:
		return nil, fmt.Errorf("unknown module: %s. Available modules: conference, scriptures, ensign", module)
	}
}
