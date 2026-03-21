// runlit check — fetch a PR diff from GitHub and eval it.
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	checkRepo        string
	checkGitHubToken string
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Fetch a GitHub PR diff and eval it",
	Long: `Fetches the diff for a pull request from GitHub and runs a full eval.

Requires a GitHub token with repo read access (GITHUB_TOKEN env var or --github-token flag).

Examples:
  runlit check --pr 42 --repo myorg/myrepo
  GITHUB_TOKEN=ghp_... runlit check --pr 7 --repo runlit-dev/api`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prStr, _ := cmd.Flags().GetString("pr")
		if prStr == "" {
			return fmt.Errorf("--pr is required")
		}
		prNumber, err := strconv.Atoi(prStr)
		if err != nil {
			return fmt.Errorf("--pr must be a number")
		}
		if checkRepo == "" {
			return fmt.Errorf("--repo is required (e.g. myorg/myrepo)")
		}

		ghToken := checkGitHubToken
		if ghToken == "" {
			ghToken = os.Getenv("GITHUB_TOKEN")
		}

		key := apiKey
		if key == "" {
			key = os.Getenv("RUNLIT_API_KEY")
		}
		if key == "" {
			key = "dev" // Phase 1: any non-empty key works
		}

		// Fetch diff from GitHub
		fmt.Fprintf(os.Stderr, "Fetching diff for %s#%d...\n", checkRepo, prNumber)
		diff, prTitle, prBody, err := fetchGitHubPRDiff(checkRepo, prNumber, ghToken)
		if err != nil {
			return fmt.Errorf("fetch diff: %w", err)
		}
		if diff == "" {
			fmt.Println("No diff found — PR may be empty or already merged.")
			return nil
		}

		// Call runlit API
		fmt.Fprintln(os.Stderr, "Running eval...")
		result, err := callEvalAPI(apiURL, key, diff, prTitle, prBody, checkRepo, int64(prNumber), readRunlitYml())
		if err != nil {
			return fmt.Errorf("eval: %w", err)
		}

		printEvalResult(result)
		return gradeExitCode(result.Grade)
	},
}

func init() {
	checkCmd.Flags().String("pr", "", "Pull request number (required)")
	checkCmd.Flags().StringVar(&checkRepo, "repo", "", "Repository full name e.g. myorg/myrepo (required)")
	checkCmd.Flags().StringVar(&checkGitHubToken, "github-token", "", "GitHub personal access token (or GITHUB_TOKEN env)")
	rootCmd.AddCommand(checkCmd)
}

func fetchGitHubPRDiff(repo string, prNumber int, token string) (diff, title, body string, err error) {
	client := &http.Client{Timeout: 15 * time.Second}

	// First fetch PR metadata (title + body)
	metaURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repo, prNumber)
	metaReq, _ := http.NewRequest(http.MethodGet, metaURL, nil)
	metaReq.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		metaReq.Header.Set("Authorization", "Bearer "+token)
	}

	metaResp, err := client.Do(metaReq)
	if err != nil {
		return "", "", "", err
	}
	defer metaResp.Body.Close()
	if metaResp.StatusCode == http.StatusNotFound {
		return "", "", "", fmt.Errorf("PR %s#%d not found (check --repo and --pr)", repo, prNumber)
	}
	if metaResp.StatusCode == http.StatusUnauthorized || metaResp.StatusCode == http.StatusForbidden {
		return "", "", "", fmt.Errorf("GitHub auth failed — set GITHUB_TOKEN or --github-token")
	}

	var pr struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	_ = json.NewDecoder(metaResp.Body).Decode(&pr)

	// Fetch diff
	diffReq, _ := http.NewRequest(http.MethodGet, metaURL, nil)
	diffReq.Header.Set("Accept", "application/vnd.github.v3.diff")
	if token != "" {
		diffReq.Header.Set("Authorization", "Bearer "+token)
	}

	diffResp, err := client.Do(diffReq)
	if err != nil {
		return "", "", "", err
	}
	defer diffResp.Body.Close()

	rawDiff, err := io.ReadAll(io.LimitReader(diffResp.Body, 1<<20))
	return string(rawDiff), pr.Title, pr.Body, err
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

type evalRequest struct {
	Diff      string `json:"diff"`
	PRTitle   string `json:"pr_title,omitempty"`
	PRDesc    string `json:"pr_description,omitempty"`
	Repo      string `json:"repo,omitempty"`
	PRNumber  int64  `json:"pr_number,omitempty"`
	RunlitYml string `json:"runlit_yml,omitempty"`
}

type signalResult struct {
	Score     float32 `json:"score"`
	Model     string  `json:"model"`
	LatencyMs int64   `json:"latency_ms"`
}

type evalResponse struct {
	EvalID        string       `json:"eval_id"`
	Score         int32        `json:"score"`
	Grade         string       `json:"grade"`
	Hallucination signalResult `json:"hallucination"`
	Intent        signalResult `json:"intent"`
	Security      signalResult `json:"security"`
	Compliance    signalResult `json:"compliance"`
	LatencyMs     int64        `json:"latency_ms"`
}

// readRunlitYml reads .runlit.yml from the current working directory.
// Returns an empty string silently if the file doesn't exist.
func readRunlitYml() string {
	b, err := os.ReadFile(".runlit.yml")
	if err != nil {
		return ""
	}
	return string(b)
}

func callEvalAPI(baseURL, key, diff, title, desc, repo string, prNumber int64, runitlYml string) (*evalResponse, error) {
	reqBody := evalRequest{
		Diff:      diff,
		PRTitle:   title,
		PRDesc:    desc,
		Repo:      repo,
		PRNumber:  prNumber,
		RunlitYml: runitlYml,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/eval", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusTooManyRequests {
		// Parse the detail field if available; otherwise use raw body.
		var errBody struct {
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(body, &errBody)
		msg := errBody.Detail
		if msg == "" {
			msg = string(body)
		}
		return nil, fmt.Errorf("quota exceeded: %s\nUpgrade at https://app.runlit.dev/billing", msg)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result evalResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}

func printEvalResult(r *evalResponse) {
	if outputFormat == "json" {
		out, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(out))
		return
	}

	gradeIcon := map[string]string{
		"PASS":  "🟢",
		"WARN":  "🟡",
		"BLOCK": "🔴",
	}[r.Grade]

	fmt.Printf("\nrunlit eval — %s %s\n", r.Grade, gradeIcon)
	fmt.Printf("Score: %d/100\n\n", r.Score)
	fmt.Printf("%-15s %-6s %s\n", "Signal", "Score", "Model")
	fmt.Printf("%-15s %-6s %s\n", "──────────────", "─────", "──────────────────────────────")
	fmt.Printf("%-15s %-6.2f %s\n", "Hallucination", r.Hallucination.Score, r.Hallucination.Model)
	fmt.Printf("%-15s %-6.2f %s\n", "Intent", r.Intent.Score, r.Intent.Model)
	fmt.Printf("%-15s %-6.2f %s\n", "Security", r.Security.Score, r.Security.Model)
	fmt.Printf("%-15s %-6.2f\n", "Compliance", r.Compliance.Score)
	fmt.Printf("\nEval ID: %s\n", r.EvalID)
	fmt.Printf("Latency: %dms\n", r.LatencyMs)
}

func gradeExitCode(grade string) error {
	if grade == "BLOCK" {
		os.Exit(1)
	}
	return nil
}
