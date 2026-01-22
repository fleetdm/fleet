package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

const url = "https://github.com/fleetdm/fleet/issues?q=is%3Aopen%20project%3Afleetdm%2F71%20label%3A%3Aproduct"

const (
	owner = "fleetdm"
	repo  = "fleet"
)

func main() {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	html := string(body)

	tickets := extractTicketNumbers(html)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		panic("GITHUB_TOKEN is not set")
	}

	err = addLabelsToTickets(token, tickets, []string{":release"})
	if err != nil {
		panic(err)
	}
}

func extractTicketNumbers(html string) []int {
	re := regexp.MustCompile(`/fleetdm/fleet/issues/(\d+)`)
	matches := re.FindAllStringSubmatch(html, -1)

	seen := map[int]bool{}
	var result []int

	for _, m := range matches {
		num, _ := strconv.Atoi(m[1])
		if seen[num] {
			continue
		}
		seen[num] = true
		result = append(result, num)
	}

	return result
}

func addLabelsToTickets(token string, issueNumbers []int, labels []string) error {
	for _, issueNumber := range issueNumbers {

		payload, err := json.Marshal(map[string][]string{
			"labels": labels,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal labels JSON: %w", err)
		}

		cmd := exec.Command("curl",
			"-s",
			"-X", "POST",
			"-H", "Authorization: Bearer "+token,
			"-H", "Accept: application/vnd.github+json",
			"-H", "Content-Type: application/json",
			"-d", string(payload),
			fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/labels", owner, repo, issueNumber),
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to add labels to issue #%d: %w, output: %s", issueNumber, err, output)
		}
	}

	return nil
}
