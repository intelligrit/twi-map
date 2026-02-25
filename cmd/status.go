package cmd

import (
	"fmt"
	"sort"

	"github.com/intelligrit/twi-map/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pipeline progress",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(dataDir)
		if err != nil {
			return err
		}
		defer s.Close()

		chapCount := s.ChapterCount()
		textCount := s.ChapterTextCount()
		extCount := s.ExtractionCount()
		locCount := s.LocationCount()

		fmt.Printf("Pipeline Status\n")
		fmt.Printf("===============\n")
		fmt.Printf("TOC chapters:    %d\n", chapCount)
		fmt.Printf("Chapters scraped: %d / %d\n", textCount, chapCount)
		fmt.Printf("Chapters extracted: %d / %d\n", extCount, chapCount)
		fmt.Printf("Aggregated locations: %d\n", locCount)

		// Per-volume breakdown
		chapByVol := s.ChapterCountByVolume()
		scrapedByVol := s.ScrapedCountByVolume()
		extractedByVol := s.ExtractedCountByVolume()

		if len(chapByVol) > 0 {
			fmt.Printf("\nPer-Volume Breakdown\n")
			fmt.Printf("--------------------\n")

			var vols []string
			for v := range chapByVol {
				vols = append(vols, v)
			}
			sort.Strings(vols)

			for _, vol := range vols {
				total := chapByVol[vol]
				scraped := scrapedByVol[vol]
				extracted := extractedByVol[vol]
				fmt.Printf("  %-8s  chapters: %3d  scraped: %3d  extracted: %3d\n",
					vol, total, scraped, extracted)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
