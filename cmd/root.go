package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags during build)
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	// Flags
	templatesDir      string
	profilePath       string
	enableTemplatesDir bool
	reloadInterval     string
)

var rootCmd = &cobra.Command{
	Use:   "claim-machinery-api",
	Short: "Claim Machinery API - Template rendering service",
	Long: `Claim Machinery API provides a service for rendering claim templates
using KCL (KusionStack Configuration Language) from OCI registries.

By default, it starts the API server. Use 'render' subcommand for the interactive template renderer.`,
	// Default behavior: run server
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&templatesDir, "templates-dir", "", "Path to templates directory")
	rootCmd.PersistentFlags().StringVar(&profilePath, "template-profile-path", "", "Path to template profile YAML")
	rootCmd.PersistentFlags().BoolVar(&enableTemplatesDir, "enable-templates-dir", false, "Enable loading templates from templates directory")
	rootCmd.PersistentFlags().StringVar(&reloadInterval, "reload-interval", "", "Profile auto-reload interval (e.g. 2m, 30s, 0 to disable)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
