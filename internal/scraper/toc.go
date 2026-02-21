package scraper

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/robertmeta/twi-map/internal/model"
)

const tocURL = "https://wanderinginn.com/table-of-contents/"

var slugRe = regexp.MustCompile(`/([^/]+)/?$`)

// ScrapeTOC fetches and parses the table of contents from the TWI website.
func ScrapeTOC() (*model.TOC, error) {
	resp, err := http.Get(tocURL)
	if err != nil {
		return nil, fmt.Errorf("fetching TOC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("TOC returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing TOC HTML: %w", err)
	}

	return ParseTOC(doc)
}

// ParseTOC extracts chapter data from a goquery document of the TOC page.
func ParseTOC(doc *goquery.Document) (*model.TOC, error) {
	var chapters []model.Chapter
	index := 0

	doc.Find(".volume-wrapper").Each(func(_ int, vol *goquery.Selection) {
		volID, _ := vol.Attr("id")
		volume := volID // e.g. "vol-1"

		vol.Find(".chapter-entry").Each(func(_ int, entry *goquery.Selection) {
			ch := model.Chapter{
				Volume: volume,
				Index:  index,
			}

			// Web title and URL
			link := entry.Find(".body-web a")
			ch.WebTitle = strings.TrimSpace(link.Text())
			ch.URL, _ = link.Attr("href")

			// Slug from URL
			if matches := slugRe.FindStringSubmatch(ch.URL); len(matches) > 1 {
				ch.Slug = matches[1]
			}

			// Book number
			if bn, exists := entry.Attr("data-book-number"); exists {
				fmt.Sscanf(bn, "%d", &ch.BookNumber)
			}

			// Audiobook chapter
			ch.AudiobookChapter = strings.TrimSpace(entry.Find(".body-audiobook").Text())

			// Ebook chapter
			ch.EbookChapter = strings.TrimSpace(entry.Find(".body-ebook").Text())

			if ch.WebTitle != "" && ch.URL != "" {
				chapters = append(chapters, ch)
				index++
			}
		})
	})

	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found in TOC")
	}

	return &model.TOC{Chapters: chapters}, nil
}
