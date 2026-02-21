package scraper

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleChapterHTML = `<html><body>
<div id="reader-content">
  <article class="twi-article">
    <p>The young woman blinked as the world spun around her.</p>
    <p>She was somewhere new. The air smelled different—like grass and something else. Something alive.</p>
    <p></p>
    <p>The inn was old. It sat on a hill, overlooking the city of Liscor far below.</p>
  </article>
</div>
<div class="comments">
  <p>This is a comment that should not be extracted.</p>
</div>
</body></html>`

func TestExtractChapterText(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleChapterHTML))
	if err != nil {
		t.Fatalf("parsing HTML: %v", err)
	}

	text := ExtractChapterText(doc)

	if !strings.Contains(text, "The young woman blinked") {
		t.Error("expected first paragraph in output")
	}
	if !strings.Contains(text, "city of Liscor") {
		t.Error("expected Liscor mention in output")
	}
	if strings.Contains(text, "comment that should not") {
		t.Error("comment text should not be in output")
	}

	// Empty paragraphs should be excluded
	paragraphs := strings.Split(text, "\n\n")
	for _, p := range paragraphs {
		if strings.TrimSpace(p) == "" {
			t.Error("empty paragraph found in output")
		}
	}
}

func TestExtractChapterText_Empty(t *testing.T) {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader("<html><body></body></html>"))
	text := ExtractChapterText(doc)
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}
