package scraper

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleTOCHTML = `<html><body>
<div class="volume-wrapper" id="vol-1">
  <div class="chapter-entry" data-book-number="1">
    <span class="body-web"><a href="https://wanderinginn.com/2016/07/27/1-00/">1.00</a></span>
    <span class="body-audiobook">Ch. 1</span>
    <span class="body-ebook">Ch. 1</span>
  </div>
  <div class="chapter-entry" data-book-number="1">
    <span class="body-web"><a href="https://wanderinginn.com/2016/07/29/1-01/">1.01</a></span>
    <span class="body-audiobook">Ch. 2</span>
    <span class="body-ebook">Ch. 2</span>
  </div>
</div>
<div class="volume-wrapper" id="vol-2">
  <div class="chapter-entry" data-book-number="2">
    <span class="body-web"><a href="https://wanderinginn.com/2017/04/22/2-00/">2.00</a></span>
    <span class="body-audiobook">Ch. 1</span>
    <span class="body-ebook">Ch. 1</span>
  </div>
</div>
</body></html>`

func TestParseTOC(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleTOCHTML))
	if err != nil {
		t.Fatalf("parsing HTML: %v", err)
	}

	toc, err := ParseTOC(doc)
	if err != nil {
		t.Fatalf("ParseTOC: %v", err)
	}

	if len(toc.Chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(toc.Chapters))
	}

	ch := toc.Chapters[0]
	if ch.WebTitle != "1.00" {
		t.Errorf("expected title '1.00', got %q", ch.WebTitle)
	}
	if ch.Volume != "vol-1" {
		t.Errorf("expected volume 'vol-1', got %q", ch.Volume)
	}
	if ch.BookNumber != 1 {
		t.Errorf("expected book number 1, got %d", ch.BookNumber)
	}
	if ch.Slug != "1-00" {
		t.Errorf("expected slug '1-00', got %q", ch.Slug)
	}
	if ch.AudiobookChapter != "Ch. 1" {
		t.Errorf("expected audiobook 'Ch. 1', got %q", ch.AudiobookChapter)
	}
	if ch.Index != 0 {
		t.Errorf("expected index 0, got %d", ch.Index)
	}

	// Second chapter
	ch2 := toc.Chapters[1]
	if ch2.Index != 1 {
		t.Errorf("expected index 1, got %d", ch2.Index)
	}

	// Third chapter (vol-2)
	ch3 := toc.Chapters[2]
	if ch3.Volume != "vol-2" {
		t.Errorf("expected volume 'vol-2', got %q", ch3.Volume)
	}
	if ch3.Index != 2 {
		t.Errorf("expected index 2, got %d", ch3.Index)
	}
}

func TestParseTOC_Empty(t *testing.T) {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader("<html><body></body></html>"))
	_, err := ParseTOC(doc)
	if err == nil {
		t.Fatal("expected error for empty TOC")
	}
}
