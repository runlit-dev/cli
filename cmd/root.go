// runlit — AI code eval CLI
// MIT License — Copyright 2026 runlit
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	apiURL       string
	apiKey       string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "runlit",
	Short: "AI code eval — catch hallucinated APIs, intent drift, and security issues in PRs",
	Long: `runlit evaluates AI-generated code diffs for:
  - Hallucinated APIs (methods that don't exist in the library)
  - Intent mismatch (code doesn't match what the PR claims to do)
  - Security vulnerabilities (hardcoded secrets, injection, etc.)
  - Compliance violations (PCI-DSS, HIPAA, SOC2)

Examples:
  runlit check --pr 42 --repo myorg/myrepo
  runlit eval --diff patch.diff --title "Add Stripe payment"
  runlit eval --diff - < patch.diff`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://api.runlit.dev", "runlit API base URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "key", "", "runlit API key (or set RUNLIT_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "text", "Output format: text or json")
}
