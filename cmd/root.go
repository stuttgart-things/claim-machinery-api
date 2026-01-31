package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	templatesDir string
	profilePath  string
)

var rootCmd = &cobra.Command{
	Use:   "claim-machinery-api",
	Short: "Claim Machinery API - Template rendering service",
	Long: `Claim Machinery API provides a service for rendering claim templates
using KCL (KusionStack Configuration Language) from OCI registries.

By default, it starts the API server. Use subcommands for other operations.`,
	// Default behavior: run server
	RunE: func(cmd *cobra.Command, args []string) error {
		return serverCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&templatesDir, "templates-dir", "", "Path to templates directory")
	rootCmd.PersistentFlags().StringVar(&profilePath, "template-profile-path", "", "Path to template profile YAML")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
