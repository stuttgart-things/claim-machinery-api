package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit SHA, and build date of claim-machinery-api.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("claim-machinery-api\n")
		fmt.Printf("  Version:    %s\n", version)
		fmt.Printf("  Commit:     %s\n", commit)
		fmt.Printf("  Build Date: %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
