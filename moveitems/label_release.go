package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	owner = "fleetdm"
	repo  = "fleet"
)

func main() {
	if len(os.Args) < 2 {
		panic("Usage: go run label_release.go <project_number>")
	}

	projectNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("Invalid project number")
	}

	urlStr := fmt.Sprintf(
		"https://github.com/fleetdm/fleet/issues?q=is%%3Aopen%%20project%%3Afleetdm%%2F%d%%20label%%3A%%3Aproduct",
		projectNumber,
	)

	resp, err := http.Get(urlStr)
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

	// Print tickets BEFORE changing anything
	fmt.Printf("Found %d ticket(s) in project %d with label :product\n", len(tickets), projectNumber)
	for _, n := range tickets {
		fmt.Printf("#%d  https://github.com/%s/%s/issues/%d\n", n, owner, repo, n)
	}

	if len(tickets) == 0 {
		return
	}

	// Ask for permission
	fmt.Print("\nProceed to remove ':product' and add ':release' to these tickets? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer != "y" && answer != "yes" {
		fmt.Println("Cancelled. No changes made.")
		return
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		panic("GITHUB_TOKEN is not set")
	}

	// remove ":product"
	if err := applyLabelsToTickets(token, tickets, []string{":product"}, false); err != nil {
		panic(err)
	}

	// add ":release"
	if err := applyLabelsToTickets(token, tickets, []string{":release"}, true); err != nil {
		panic(err)
	}

	fmt.Println("Done.")
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

// add=true  -> add labels
// add=false -> remove labels
func applyLabelsToTickets(token string, issueNumbers []int, labels []string, add bool) error {
	for _, issueNumber := range issueNumbers {
		if add {
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

		} else {
			for _, label := range labels {
				escaped := url.PathEscape(label)

				cmd := exec.Command("curl",
					"-s",
					"-X", "DELETE",
					"-H", "Authorization: Bearer "+token,
					"-H", "Accept: application/vnd.github+json",
					fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/labels/%s", owner, repo, issueNumber, escaped),
				)

				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to remove label %q from issue #%d: %w, output: %s", label, issueNumber, err, output)
				}
			}
		}
	}

	return nil
}
