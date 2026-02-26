package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/intelligrit/twi-map/internal/scraper"
	"github.com/intelligrit/twi-map/internal/store"
	"github.com/spf13/cobra"
)

var scrapeChaptersVolume string

var scrapeChaptersCmd = &cobra.Command{
	Use:   "scrape-chapters",
	Short: "Download chapter plaintext (cached, rate-limited)",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(dataDir)
		if err != nil {
			return err
		}
		defer s.Close()

		toc, err := s.ReadTOC()
		if err != nil {
			return fmt.Errorf("reading TOC (run scrape-toc first): %w", err)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		rl := scraper.NewRateLimiter(cfg.Scrape.RateLimit)

		var toScrape []int
		for _, ch := range toc.Chapters {
			if scrapeChaptersVolume != "" && ch.Volume != scrapeChaptersVolume {
				continue
			}
			if !s.ChapterTextExists(ch.Index) {
				toScrape = append(toScrape, ch.Index)
			}
		}

		if len(toScrape) == 0 {
			fmt.Println("All matching chapters already scraped.")
			return nil
		}

		fmt.Printf("Scraping %d chapters...\n", len(toScrape))

		for i, idx := range toScrape {
			ch := toc.Chapters[idx]

			select {
			case <-ctx.Done():
				fmt.Printf("\nInterrupted after %d/%d chapters\n", i, len(toScrape))
				return nil
			default:
			}

			logVerbose("  [%d/%d] %s", i+1, len(toScrape), ch.WebTitle)

			text, err := scraper.ScrapeChapter(ctx, ch.URL, rl)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  WARNING: failed to scrape %q: %v\n", ch.WebTitle, err)
				continue
			}

			if err := s.WriteChapterText(ch.Index, text); err != nil {
				return fmt.Errorf("saving chapter text: %w", err)
			}

			fmt.Printf("  [%d/%d] %s (%d chars)\n", i+1, len(toScrape), ch.WebTitle, len(text))
		}

		fmt.Println("Done.")
		return nil
	},
}

func init() {
	scrapeChaptersCmd.Flags().StringVar(&scrapeChaptersVolume, "volume", "", "Only scrape chapters from this volume (e.g. vol-1)")
	rootCmd.AddCommand(scrapeChaptersCmd)
}
