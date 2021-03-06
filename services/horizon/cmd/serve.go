package cmd

import (
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "run horizon server",
	Long:  "serve initializes then starts the horizon HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		initApp().Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
