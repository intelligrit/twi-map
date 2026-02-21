package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dataDir string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "twi-map",
	Short: "Extract location data from The Wandering Inn and generate interactive maps",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "data", "Directory for storing scraped/extracted data")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}

func logVerbose(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
