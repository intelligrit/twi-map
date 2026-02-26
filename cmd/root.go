package cmd

import (
	"fmt"
	"os"

	"github.com/intelligrit/twi-map/internal/config"
	"github.com/spf13/cobra"
)

var (
	dataDir    string
	verbose    bool
	configPath string
	cfg        *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "twi-map",
	Short: "Extract location data from The Wandering Inn and generate interactive maps",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if !cmd.Flags().Changed("data-dir") {
			dataDir = cfg.Data.Dir
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.toml", "Path to configuration file")
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
