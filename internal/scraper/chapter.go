package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ScrapeChapter fetches a chapter URL and extracts the plaintext content.
func ScrapeChapter(ctx context.Context, url string, rl *RateLimiter) (string, error) {
	if rl != nil {
		if err := rl.Wait(ctx); err != nil {
			return "", fmt.Errorf("rate limiter: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching chapter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("chapter returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parsing chapter HTML: %w", err)
	}

	return ExtractChapterText(doc), nil
}

// ExtractChapterText pulls plaintext from a chapter's goquery document.
func ExtractChapterText(doc *goquery.Document) string {
	var paragraphs []string

	doc.Find("#reader-content article.twi-article p").Each(func(_ int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})

	return strings.Join(paragraphs, "\n\n")
}
