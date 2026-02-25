package cmd

import (
	"github.com/intelligrit/twi-map/internal/store"
	"github.com/intelligrit/twi-map/internal/web"
	"github.com/spf13/cobra"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the interactive map web app",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(dataDir)
		if err != nil {
			return err
		}
		defer s.Close()

		srv := &web.Server{
			Store: s,
			Addr:  serveAddr,
		}
		return srv.ListenAndServe()
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "localhost:8080", "Address to listen on")
	rootCmd.AddCommand(serveCmd)
}
