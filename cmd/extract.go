package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/robertmeta/twi-map/internal/extractor"
	"github.com/robertmeta/twi-map/internal/model"
	"github.com/robertmeta/twi-map/internal/store"
	"github.com/spf13/cobra"
)

var (
	extractVolume string
	extractModel  string
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract location data from chapter text using Claude API",
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

		client, err := extractor.NewClient(extractModel)
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		var toExtract []model.Chapter
		for _, ch := range toc.Chapters {
			if extractVolume != "" && ch.Volume != extractVolume {
				continue
			}
			if !s.ChapterTextExists(ch.Index) {
				continue // skip chapters we haven't scraped
			}
			if s.ExtractionExists(ch.Index) {
				continue // skip already extracted
			}
			toExtract = append(toExtract, ch)
		}

		if len(toExtract) == 0 {
			fmt.Println("All matching chapters already extracted.")
			return nil
		}

		fmt.Printf("Extracting locations from %d chapters using %s...\n", len(toExtract), extractModel)

		var totalInput, totalOutput int
		for i, ch := range toExtract {
			select {
			case <-ctx.Done():
				fmt.Printf("\nInterrupted after %d/%d chapters\n", i, len(toExtract))
				return nil
			default:
			}

			text, err := s.ReadChapterText(ch.Index)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  WARNING: failed to read chapter %d: %v\n", ch.Index, err)
				continue
			}

			fmt.Printf("  [%d/%d] %s (%d chars)...", i+1, len(toExtract), ch.WebTitle, len(text))

			rawResp, usage, err := client.Extract(ctx, ch.WebTitle, text)
			if err != nil {
				fmt.Fprintf(os.Stderr, " ERROR: %v\n", err)
				continue
			}

			totalInput += usage.InputTokens
			totalOutput += usage.OutputTokens

			parsed, err := extractor.ParseExtraction(rawResp)
			if err != nil {
				fmt.Fprintf(os.Stderr, " PARSE ERROR: %v\n", err)
				continue
			}

			ext := &model.ChapterExtraction{
				ChapterIndex:  ch.Index,
				ChapterTitle:  ch.WebTitle,
				Locations:     parsed.Locations,
				Relationships: parsed.Relationships,
				Containment:   parsed.Containment,
				Model:         extractModel,
				ExtractedAt:   time.Now().UTC().Format(time.RFC3339),
			}

			if err := s.WriteExtraction(ext); err != nil {
				return fmt.Errorf("saving extraction: %w", err)
			}

			fmt.Printf(" %d locations, %d relationships (%d+%d tokens)\n",
				len(parsed.Locations), len(parsed.Relationships),
				usage.InputTokens, usage.OutputTokens)
		}

		fmt.Printf("\nDone. Total tokens: %d input, %d output\n", totalInput, totalOutput)
		return nil
	},
}

func init() {
	extractCmd.Flags().StringVar(&extractVolume, "volume", "", "Only extract from this volume (e.g. vol-1)")
	extractCmd.Flags().StringVar(&extractModel, "model", "claude-sonnet-4-20250514", "Anthropic model to use")
	rootCmd.AddCommand(extractCmd)
}
