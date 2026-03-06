package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/dropseed/git-stats/internal/stats"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func ReportStatus(key string, currentValue float64, previousValue float64, hasPrevious bool, goal string, stat *stats.CommitStat) error {
	token := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")
	sha := os.Getenv("GITHUB_SHA")

	if token == "" || repo == "" || sha == "" {
		return nil
	}

	state := "success"
	description := stat.FormatValue(currentValue)

	if goal != "" && hasPrevious {
		switch goal {
		case "increase":
			if currentValue < previousValue {
				state = "failure"
				description = fmt.Sprintf("%s (was %s, goal: increase)", stat.FormatValue(currentValue), stat.FormatValue(previousValue))
			} else if currentValue > previousValue {
				description = fmt.Sprintf("%s (was %s)", stat.FormatValue(currentValue), stat.FormatValue(previousValue))
			}
		case "decrease":
			if currentValue > previousValue {
				state = "failure"
				description = fmt.Sprintf("%s (was %s, goal: decrease)", stat.FormatValue(currentValue), stat.FormatValue(previousValue))
			} else if currentValue < previousValue {
				description = fmt.Sprintf("%s (was %s)", stat.FormatValue(currentValue), stat.FormatValue(previousValue))
			}
		}
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", repo, sha)

	body := map[string]string{
		"state":       state,
		"description": description,
		"context":     fmt.Sprintf("git-stats/%s", key),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting GitHub status: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub status API returned %d", resp.StatusCode)
	}

	return nil
}

func IsCI() bool {
	return os.Getenv("CI") != ""
}
