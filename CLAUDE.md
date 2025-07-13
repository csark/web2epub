# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**web2epub** is a Go command-line tool that converts web content into EPUB format ebooks. It crawls webpages, extracts content, downloads images, and packages everything into a properly formatted EPUB file.

## Commands

### Build and Run
```bash
# Run directly with Go
go run main.go -url "URL" [flags]

# Build binary
go build -o web2epub main.go

# Run built binary
./web2epub -url "URL" [flags]
```

### Command Line Flags
- `-url` (required): Starting URL to crawl
- `-output` (optional): Custom output filename (defaults to page title)
- `-cover` (optional): Cover image URL
- `-module` (optional): Collection module to use - "conference", "scriptures", "ensign" (default: "conference")
- `-same-host` (optional): Restrict crawling to same domain (default: true)

### Testing Commands
```bash
# Test with conference listing (default module)
go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2025/04?lang=eng" -cover "https://www.churchofjesuschrist.org/imgs/1hsgsztu81qrzo6i6u7l0g7d8zvop7dg93ctfug4/full/%21320%2C/0/default"

# Test with specific module
go run main.go -url "URL" -module "conference" -output "my-ebook"

# Test different modules
go run main.go -url "URL" -module "scriptures"
go run main.go -url "URL" -module "ensign"
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

## Architecture

### Modular Structure
The application now uses a modular architecture with:

1. **Main Application** (`main.go`): Handles CLI flags, coordinates modules, generates EPUB
2. **Collectors Package** (`collectors/`): Modular content collection system
   - `types.go`: Common types and configuration structures
   - `link_collector.go`: Discovers links from starting pages
   - `page_collector.go`: Extracts content from discovered pages
3. **Module Configurations**: Pre-configured settings for different site sections

### Key Dependencies
- `github.com/gocolly/colly/v2`: Web scraping and crawling
- `github.com/PuerkitoBio/goquery`: HTML parsing and jQuery-like selection
- `github.com/go-shiori/go-epub`: EPUB creation and management

### Processing Flow
1. Parse command line flags and validate URL
2. Load appropriate collector configuration based on `-module` flag
3. Create temporary directory for resources
4. **Link Discovery**: Use modular link collector with config-specific selectors
5. **Content Extraction**: Process pages in parallel using config-specific content selectors
6. **Content Processing**: Clean content, download images, determine section structure
7. Generate EPUB with custom CSS styling

### Available Modules
- **conference**: General Conference talks (`.list-tile` links, `article` content)
- **scriptures**: Scripture pages (all links, `.body-block` content)  
- **ensign**: Ensign magazine articles (`.title-link` links, `article` content)

### Special Features
- **Church Content Optimization**: Specific handling for LDS Church website content with author name cleaning
- **Parallel Processing**: Concurrent page processing with configurable delays
- **Image Handling**: Automatic image downloading and embedding with proper MIME type detection
- **Section Management**: Intelligent organization into sections and subsections based on content length (>2000 chars = section)

### Content Extraction
- Targets `article` elements for main content
- Extracts titles from `h1` tags
- Processes author information with Church-specific cleaning
- Downloads and embeds all images found in content
- Applies custom CSS for proper ebook formatting