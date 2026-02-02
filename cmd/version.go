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
		fmt.Println(logo)
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Commit:     %s\n", Commit)
		fmt.Printf("Build Date: %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
