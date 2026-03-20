package main

import (
	"embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed llm_docs/*.md
var llmDocs embed.FS

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "LLM agent documentation commands",
}

var llmOnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Print onboarding context for LLM agents",
	Long:  "Dumps structured onboarding documentation that teaches an LLM agent how to use the portfolio tool.",
	Run: func(cmd *cobra.Command, args []string) {
		content, err := llmDocs.ReadFile("llm_docs/onboard.md")
		if err != nil {
			fmt.Printf("Error reading onboard docs: %v\n", err)
			return
		}
		fmt.Print(string(content))
	},
}

func init() {
	llmCmd.AddCommand(llmOnboardCmd)
	rootCmd.AddCommand(llmCmd)
}
