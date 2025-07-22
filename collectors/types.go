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
	URL          string
	IsSubSection bool
	Order        int
}

type StringPair struct {
	OldText      string
	NewText      string
	IsSubSection bool
}

// CollectorConfig holds configuration for different collection strategies
type CollectorConfig struct {
	CollectorType string

	// Link discovery settings
	LinkSelector string       // CSS selector for finding links
	LinkFilter   string       // String to match in a url
	LinkReplace  []StringPair // Strings that define a part of a url to replace

	TitleSelector   string   // CSS selector for page title
	AuthorSelector  string   // CSS selector for author
	ContentSelector string   // CSS selector for main content
	RemoveSelectors []string // CSS selectors for elements to remove
	UnwrapSelectors []string // CSS selectors for elements to unwrap (e.g. keep text but remove html tags around the text)

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
	CollectorCSS   string   // The css to include for the collector type
}

// GetGeneralConferenceConfig returns config for LDS General Conference pages
func GetGeneralConferenceConfig() *CollectorConfig {
	return &CollectorConfig{
		CollectorType:   "conference",
		LinkSelector:    "a[href].list-tile",
		LinkFilter:      "sassafrass",
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
		CollectorCSS: `h1 {
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
    margin-block-end: 0.75em;
}
p.kicker {
    font-style: italic;
	margin-block-start: 1em;
    margin-block-end: 2em;
}`,
	}
}

// GetScripturesConfig returns config for LDS Scripture pages
func GetScripturesConfig() *CollectorConfig {
	return &CollectorConfig{
		CollectorType: "scriptures",
		LinkSelector:  "a[href].list-tile",
		LinkFilter:    "illustrations",
		LinkReplace: []StringPair{
			{"/_contents", "", true},
		},
		TitleSelector:   "title",
		AuthorSelector:  "",
		ContentSelector: ".body",
		RemoveSelectors: []string{
			"script", "footer", "iframe", "button",
			".nav", ".menu", ".sidebar", ".ad", ".ads",
			".study-notes", ".footnotes",
		},
		UnwrapSelectors: []string{
			".study-note-ref",
		},
		AuthorReplacements:  map[string]string{},
		DefaultAuthor:       "",
		SubSectionThreshold: 30,
		FallbackToBody:      true,
		Parallelism:         10,
		DelaySeconds:        5,
		SkipExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif",
			".pdf", ".zip", ".mp3", ".mp4",
		},
		CollectorCSS: `.title-number {
    font-weight: bold;
}
.study-summary {
    font-style: italic;
}
.title-number {
	display: block;
    font-size: 1.5em;
    margin-block-start: 0.83em;
    margin-block-end: 0.83em;
    margin-inline-start: 0px;
    margin-inline-end: 0px;
    font-weight: bold;
}`,
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
