// runlit status — show current plan and eval usage.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type usageResponse struct {
	TenantID   string `json:"tenant_id"`
	YearMonth  string `json:"year_month"`
	EvalsUsed  int    `json:"evals_used"`
	EvalsLimit int    `json:"evals_limit"`
	Plan       string `json:"plan"`
	ResetsAt   string `json:"resets_at"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current plan and eval usage",
	Long: `Displays your current runlit plan, monthly eval usage, and quota limits.

Examples:
  runlit status
  runlit status --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		key := apiKey
		if key == "" {
			key = os.Getenv("RUNLIT_API_KEY")
		}
		if key == "" {
			return fmt.Errorf("API key required — set RUNLIT_API_KEY or use --key")
		}

		usage, err := callUsageAPI(apiURL, key)
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}

		if outputFormat == "json" {
			out, _ := json.MarshalIndent(usage, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		printStatus(usage)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func callUsageAPI(baseURL, key string) (*usageResponse, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var usage usageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &usage, nil
}

func printStatus(u *usageResponse) {
	const width = 30

	fmt.Printf("\nrunlit status — %s\n\n", u.YearMonth)
	fmt.Printf("  Plan:     %s\n", u.Plan)

	if u.EvalsLimit == 0 {
		fmt.Printf("  Evals:    %d / unlimited\n", u.EvalsUsed)
	} else {
		pct := float64(u.EvalsUsed) / float64(u.EvalsLimit) * 100
		filled := int(pct / 100 * float64(width))
		if filled > width {
			filled = width
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		fmt.Printf("  Evals:    %d / %d (%.0f%%)\n", u.EvalsUsed, u.EvalsLimit, pct)
		fmt.Printf("            [%s]\n", bar)

		if pct >= 95 {
			fmt.Printf("\n  ⚠  Quota critical — upgrade to avoid interruption.\n")
			fmt.Printf("  →  https://app.runlit.dev/billing\n")
		} else if pct >= 80 {
			fmt.Printf("\n  ⚠  Approaching quota limit.\n")
			fmt.Printf("  →  https://app.runlit.dev/billing\n")
		}
	}

	if u.ResetsAt != "" {
		if t, err := time.Parse(time.RFC3339, u.ResetsAt); err == nil {
			fmt.Printf("  Resets:   %s\n", t.UTC().Format("2006-01-02"))
		}
	}
	fmt.Println()
}
