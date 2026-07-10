package ghapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GetPRStates returns the state (OPEN/CLOSED/MERGED) of many PRs in one batched
// GraphQL query (chunked), keyed by PR number. Missing/errored numbers are absent.
func GetPRStates(repo string, numbers []int) (map[int]string, error) {
	return batchStates(repo, numbers, "pullRequest")
}

// GetIssueStates returns the state (OPEN/CLOSED) of many issues, batched by number.
func GetIssueStates(repo string, numbers []int) (map[int]string, error) {
	return batchStates(repo, numbers, "issue")
}

// batchStates fetches the `state` of many PRs or issues via aliased GraphQL
// fields, 50 per request.
func batchStates(repo string, numbers []int, field string) (map[int]string, error) {
	owner, name := splitRepo(repo)
	out := map[int]string{}
	for start := 0; start < len(numbers); start += 50 {
		end := start + 50
		if end > len(numbers) {
			end = len(numbers)
		}
		chunk := numbers[start:end]

		var q strings.Builder
		q.WriteString("query($o:String!,$n:String!){repository(owner:$o,name:$n){")
		for i, num := range chunk {
			fmt.Fprintf(&q, "a%d: %s(number:%d){state} ", i, field, num)
		}
		q.WriteString("}}")

		cmd := fmt.Sprintf("gh api graphql -f query='%s' -f o='%s' -f n='%s'", q.String(), owner, name)
		res, err := RunCommandWithRetry(cmd, 3)
		if err != nil {
			return out, err
		}
		var resp struct {
			Data struct {
				Repository map[string]*struct {
					State string `json:"state"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res, &resp); err != nil {
			return out, err
		}
		for i, num := range chunk {
			if node := resp.Data.Repository[fmt.Sprintf("a%d", i)]; node != nil {
				out[num] = node.State
			}
		}
	}
	return out, nil
}
