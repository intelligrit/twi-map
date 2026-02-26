package cmd

import (
	"fmt"

	"github.com/intelligrit/twi-map/internal/store"
	"github.com/intelligrit/twi-map/internal/web"
	"github.com/spf13/cobra"
)

var (
	serveHost string
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the interactive map web app",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("host") {
			serveHost = cfg.Server.Host
		}
		if !cmd.Flags().Changed("port") {
			servePort = cfg.Server.Port
		}

		s, err := store.New(dataDir)
		if err != nil {
			return err
		}
		defer s.Close()

		srv := &web.Server{
			Store: s,
			Addr:  fmt.Sprintf("%s:%d", serveHost, servePort),
		}
		return srv.ListenAndServe()
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "Host to listen on")
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
