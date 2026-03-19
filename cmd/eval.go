// runlit eval — eval a diff from a file or stdin.
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Eval a diff from a file or stdin",
	Long: `Reads a unified diff and runs a full runlit eval.

Use '-' as the diff file to read from stdin.

Examples:
  runlit eval --diff patch.diff
  runlit eval --diff patch.diff --title "Add Stripe checkout" --repo myorg/myrepo --pr 12
  git diff HEAD~1 | runlit eval --diff -`,
	RunE: func(cmd *cobra.Command, args []string) error {
		diffFile, _ := cmd.Flags().GetString("diff")
		if diffFile == "" {
			return fmt.Errorf("--diff is required (use '-' for stdin)")
		}

		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("description")
		repo, _ := cmd.Flags().GetString("repo")
		prNumber, _ := cmd.Flags().GetInt64("pr")

		key := apiKey
		if key == "" {
			key = os.Getenv("RUNLIT_API_KEY")
		}
		if key == "" {
			key = "dev"
		}

		// Read diff from file or stdin
		var diff []byte
		var err error
		if diffFile == "-" {
			diff, err = io.ReadAll(os.Stdin)
		} else {
			diff, err = os.ReadFile(diffFile)
		}
		if err != nil {
			return fmt.Errorf("read diff: %w", err)
		}
		if len(diff) == 0 {
			return fmt.Errorf("diff is empty")
		}

		fmt.Fprintln(os.Stderr, "Running eval...")
		result, err := callEvalAPI(apiURL, key, string(diff), title, desc, repo, prNumber)
		if err != nil {
			return fmt.Errorf("eval: %w", err)
		}

		printEvalResult(result)
		return gradeExitCode(result.Grade)
	},
}

func init() {
	evalCmd.Flags().String("diff", "", "Path to unified diff file, or '-' for stdin (required)")
	evalCmd.Flags().String("title", "", "PR title (improves intent signal accuracy)")
	evalCmd.Flags().String("description", "", "PR description (improves intent signal accuracy)")
	evalCmd.Flags().String("repo", "", "Repository full name e.g. myorg/myrepo")
	evalCmd.Flags().Int64("pr", 0, "Pull request number")
	rootCmd.AddCommand(evalCmd)
}
