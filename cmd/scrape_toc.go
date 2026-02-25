package cmd

import (
	"fmt"
	"time"

	"github.com/intelligrit/twi-map/internal/scraper"
	"github.com/intelligrit/twi-map/internal/store"
	"github.com/spf13/cobra"
)

var scrapeTocCmd = &cobra.Command{
	Use:   "scrape-toc",
	Short: "Fetch and parse the table of contents from wanderinginn.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(dataDir)
		if err != nil {
			return err
		}
		defer s.Close()

		fmt.Println("Fetching table of contents...")
		toc, err := scraper.ScrapeTOC()
		if err != nil {
			return fmt.Errorf("scraping TOC: %w", err)
		}

		toc.ScrapedAt = time.Now().UTC().Format(time.RFC3339)
		if err := s.WriteTOC(toc); err != nil {
			return fmt.Errorf("saving TOC: %w", err)
		}

		fmt.Printf("Saved %d chapters across volumes\n", len(toc.Chapters))

		// Print volume breakdown
		volumes := make(map[string]int)
		for _, ch := range toc.Chapters {
			volumes[ch.Volume]++
		}
		for vol, count := range volumes {
			fmt.Printf("  %s: %d chapters\n", vol, count)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scrapeTocCmd)
}
