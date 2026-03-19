package main

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed llm_docs/*.md
var llmDocs embed.FS

//go:embed llm_docs/topics/*.md
var llmTopics embed.FS

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "LLM agent documentation commands",
}

var llmOnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Print onboarding context for LLM agents",
	Long:  "Dumps structured onboarding documentation that teaches an LLM agent how to use this CLI.",
	Run: func(cmd *cobra.Command, args []string) {
		content, err := llmDocs.ReadFile("llm_docs/onboard.md")
		if err != nil {
			fmt.Printf("Error reading onboard docs: %v\n", err)
			return
		}
		fmt.Print(string(content))
	},
}

var llmTopicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "List available deep-dive topics",
	Run: func(cmd *cobra.Command, args []string) {
		entries, err := llmTopics.ReadDir("llm_docs/topics")
		if err != nil {
			fmt.Printf("Error reading topics: %v\n", err)
			return
		}
		fmt.Println("Available topics:")
		fmt.Println()
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".md" {
				name := strings.TrimSuffix(e.Name(), ".md")
				// Read first line (after # header) for description
				content, err := llmTopics.ReadFile("llm_docs/topics/" + e.Name())
				if err != nil {
					continue
				}
				desc := extractTitle(string(content))
				fmt.Printf("  %-20s %s\n", name, desc)
			}
		}
		fmt.Println()
		fmt.Println("Run: near-intents llm topic <name>")
	},
}

var llmTopicCmd = &cobra.Command{
	Use:   "topic [name]",
	Short: "Print deep-dive documentation on a specific topic",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		filename := "llm_docs/topics/" + name + ".md"
		content, err := llmTopics.ReadFile(filename)
		if err != nil {
			// List available topics on error
			entries, _ := llmTopics.ReadDir("llm_docs/topics")
			var names []string
			for _, e := range entries {
				if filepath.Ext(e.Name()) == ".md" {
					names = append(names, strings.TrimSuffix(e.Name(), ".md"))
				}
			}
			fmt.Printf("Topic %q not found. Available topics: %s\n", name, strings.Join(names, ", "))
			return
		}
		fmt.Print(string(content))
	},
}

// extractTitle pulls the first markdown heading from content.
func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func init() {
	llmCmd.AddCommand(llmOnboardCmd)
	llmCmd.AddCommand(llmTopicsCmd)
	llmCmd.AddCommand(llmTopicCmd)
	rootCmd.AddCommand(llmCmd)
}
