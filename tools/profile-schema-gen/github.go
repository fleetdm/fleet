package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const (
	githubAPIURL = "https://api.github.com"
	codeloadURL  = "https://codeload.github.com"
)

type githubCommit struct {
	SHA string
	// Subject is the first line of the commit message.
	Subject string
}

// resolveCommit returns the commit at the head of branch. It's the only api.github.com call per repo
// (60/hour unauthenticated); tarballs come from codeload, which isn't API rate limited.
func resolveCommit(ctx context.Context, logger *slog.Logger, client *http.Client, apiURL, repo, branch string) (githubCommit, error) {
	url := fmt.Sprintf("%s/repos/%s/commits/%s", apiURL, repo, branch)
	logger.InfoContext(ctx, "resolving branch head", "repo", repo, "branch", branch, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubCommit{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return githubCommit{}, fmt.Errorf("resolve %s@%s: %w", repo, branch, err)
	}
	defer resp.Body.Close()
	logger.DebugContext(ctx, "GitHub API response", "status", resp.StatusCode,
		"ratelimit_remaining", resp.Header.Get("X-RateLimit-Remaining"), "ratelimit_reset", resp.Header.Get("X-RateLimit-Reset"))
	if err := checkStatus(resp); err != nil {
		return githubCommit{}, fmt.Errorf("resolve %s@%s: %w", repo, branch, err)
	}
	var body struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}
	// The response lists changed files too; cap it so a huge commit can't exhaust memory.
	if err := json.NewDecoder(io.LimitReader(resp.Body, 50<<20)).Decode(&body); err != nil {
		return githubCommit{}, fmt.Errorf("resolve %s@%s: decode response: %w", repo, branch, err)
	}
	if len(body.SHA) != 40 || strings.Trim(body.SHA, "0123456789abcdef") != "" {
		return githubCommit{}, fmt.Errorf("resolve %s@%s: unexpected commit SHA %q", repo, branch, body.SHA)
	}
	subject, _, _ := strings.Cut(body.Commit.Message, "\n")
	commit := githubCommit{SHA: body.SHA, Subject: strings.TrimSpace(subject)}
	logger.InfoContext(ctx, "resolved branch head", "repo", repo, "branch", branch, "commit", commit.SHA, "subject", commit.Subject)
	return commit, nil
}

func checkStatus(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusTooManyRequests,
		resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0":
		return fmt.Errorf("GitHub rate limit hit (status %d, resets at unix time %s); try again later",
			resp.StatusCode, resp.Header.Get("X-RateLimit-Reset"))
	default:
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
}
