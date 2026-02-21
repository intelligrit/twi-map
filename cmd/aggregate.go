package cmd

import (
	"fmt"

	"github.com/robertmeta/twi-map/internal/aggregator"
	"github.com/robertmeta/twi-map/internal/store"
	"github.com/spf13/cobra"
)

var aggregateCoords bool

var aggregateCmd = &cobra.Command{
	Use:   "aggregate",
	Short: "Merge per-chapter extractions into unified location dataset",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(dataDir)
		if err != nil {
			return err
		}
		defer s.Close()

		fmt.Println("Aggregating extractions...")
		data, err := aggregator.Aggregate(s)
		if err != nil {
			return fmt.Errorf("aggregation failed: %w", err)
		}

		if err := s.WriteAggregated(data); err != nil {
			return fmt.Errorf("saving aggregated data: %w", err)
		}

		fmt.Printf("Aggregated: %d locations, %d relationships, %d containment rules\n",
			len(data.Locations), len(data.Relationships), len(data.Containment))

		if aggregateCoords {
			fmt.Println("Assigning coordinates...")
			if err := aggregator.AssignCoordinates(s, data); err != nil {
				return fmt.Errorf("assigning coordinates: %w", err)
			}
			fmt.Println("Coordinates assigned.")
		}

		return nil
	},
}

func init() {
	aggregateCmd.Flags().BoolVar(&aggregateCoords, "coords", true, "Assign estimated coordinates to locations")
	rootCmd.AddCommand(aggregateCmd)
}
