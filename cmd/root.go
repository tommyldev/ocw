package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ocw",
	Short: "OCW - Open Code Workspace",
	Long:  "OCW is a terminal-based workspace manager for open source development",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("OCW v0.1.0-dev")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
