package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	Long:  "List the available models you can use with the --model flag.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("o4-mini")
		fmt.Println("gpt-3.5-turbo")
		fmt.Println("codex-cli")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
