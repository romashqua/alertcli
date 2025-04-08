package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	alertmanagerURL string
	apiVersion      string
)

var rootCmd = &cobra.Command{
	Use:   "alertctl",
	Short: "Alertmanager CLI Tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if apiVersion != "v1" && apiVersion != "v2" {
			cmd.PrintErrf("Invalid API version: %s. Use 'v1' or 'v2'\n", apiVersion)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&alertmanagerURL, "url", "u", "http://localhost:9093", "Alertmanager URL")
	rootCmd.PersistentFlags().StringVarP(&apiVersion, "api-version", "a", "v2", "API version (v1 or v2)")

	rootCmd.AddCommand(
		NewAlertsCmd(),
		NewSilencesCmd(),
		NewCompletionCmd(),
	)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
