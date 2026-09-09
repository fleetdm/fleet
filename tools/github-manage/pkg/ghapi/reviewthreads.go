package ghapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

func splitRepo(repo string) (owner, name string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", repo
	}
	return parts[0], parts[1]
}

// GetUnresolvedReviewThreadCount returns the number of unresolved review threads
// on a PR — a precise "your move" signal that reviewDecision does not capture.
func GetUnresolvedReviewThreadCount(repo string, number int) (int, error) {
	owner, name := splitRepo(repo)
	q := fmt.Sprintf(
		`query{repository(owner:"%s",name:"%s"){pullRequest(number:%d){reviewThreads(first:100){nodes{isResolved}}}}}`,
		owner, name, number,
	)
	out, err := RunGH("api", "graphql", "-f", "query="+q)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, err
	}
	count := 0
	for _, n := range resp.Data.Repository.PullRequest.ReviewThreads.Nodes {
		if !n.IsResolved {
			count++
		}
	}
	return count, nil
}
