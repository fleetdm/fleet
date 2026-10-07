package ghapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// statusBatchSize is how many issues are fetched per aliased GraphQL request.
// 40 aliases × projectItems(first:30) stays far under GitHub's node limits
// while cutting a 762-issue board from 762 requests to ~20.
const statusBatchSize = 40

// GetIssueProjectStatusesBatch returns, for many issues of one repo, every
// project each issue belongs to with its Status column — the batched
// equivalent of calling GetAllIssueProjectStatuses per issue, using aliased
// GraphQL queries (statusBatchSize issues per request). Best-effort: a chunk
// whose batched request yields nothing falls back to per-issue lookups, and
// issues that still fail are simply absent from the result. tick, when
// non-nil, is called as issues complete with (done, total).
func GetIssueProjectStatusesBatch(repoFullName string, issueNumbers []int, tick func(done, total int)) map[int]map[int]ProjectStatus {
	results := make(map[int]map[int]ProjectStatus, len(issueNumbers))
	owner, name, err := splitRepoFullName(repoFullName)
	if err != nil {
		return results
	}
	done := 0
	step := func(n int) {
		done += n
		if tick != nil {
			tick(done, len(issueNumbers))
		}
	}
	for start := 0; start < len(issueNumbers); start += statusBatchSize {
		chunk := issueNumbers[start:min(start+statusBatchSize, len(issueNumbers))]
		batch, ok := fetchStatusChunk(owner, name, chunk)
		if !ok {
			// The whole batched request failed — recover issue by issue.
			for _, num := range chunk {
				if found, err := GetAllIssueProjectStatuses(repoFullName, num); err == nil {
					results[num] = found
				}
				step(1)
			}
			continue
		}
		for num, found := range batch {
			results[num] = found
		}
		step(len(chunk))
	}
	return results
}

// fetchStatusChunk runs one aliased GraphQL request covering a chunk of
// issues. ok is false when no usable data came back (caller falls back to
// per-issue lookups); individually missing issues (null aliases, e.g. a
// deleted issue) are skipped while the rest of the chunk is kept.
func fetchStatusChunk(owner, name string, numbers []int) (map[int]map[int]ProjectStatus, bool) {
	var q strings.Builder
	q.WriteString("query($o:String!,$n:String!){repository(owner:$o,name:$n){")
	for i, num := range numbers {
		fmt.Fprintf(&q, "i%d: issue(number:%d){...IssueProjectStatuses} ", i, num)
	}
	q.WriteString("}}\n")
	q.WriteString(`fragment IssueProjectStatuses on Issue {
		projectItems(first:30){
			nodes{
				project{ number title updatedAt }
				fieldValueByName(name:"Status"){
					... on ProjectV2ItemFieldSingleSelectValue{ name }
				}
			}
		}
	}`)

	out, err := RunGHWithRetry(3, "api", "graphql", "-f", "query="+q.String(), "-f", "o="+owner, "-f", "n="+name)
	return parseStatusChunk(out, err, numbers)
}

// parseStatusChunk decodes one aliased-statuses response. ghErr is the gh
// command's error, tolerated as long as the response still carries data.
func parseStatusChunk(out []byte, ghErr error, numbers []int) (map[int]map[int]ProjectStatus, bool) {
	var resp struct {
		Data struct {
			Repository map[string]*struct {
				ProjectItems struct {
					Nodes []struct {
						Project struct {
							Number    int    `json:"number"`
							Title     string `json:"title"`
							UpdatedAt string `json:"updatedAt"`
						} `json:"project"`
						FieldValueByName *struct {
							Name string `json:"name"`
						} `json:"fieldValueByName"`
					} `json:"nodes"`
				} `json:"projectItems"`
			} `json:"repository"`
		} `json:"data"`
	}
	// gh exits non-zero on partial GraphQL errors (e.g. one alias NOT_FOUND)
	// while still printing the response, with stderr mixed in — decode the
	// leading JSON document and keep whatever data came back.
	if derr := json.NewDecoder(bytes.NewReader(out)).Decode(&resp); derr != nil {
		return nil, false
	}
	if ghErr != nil && len(resp.Data.Repository) == 0 {
		return nil, false
	}

	res := make(map[int]map[int]ProjectStatus, len(numbers))
	for i, num := range numbers {
		node := resp.Data.Repository[fmt.Sprintf("i%d", i)]
		if node == nil {
			continue
		}
		found := make(map[int]ProjectStatus, len(node.ProjectItems.Nodes))
		for _, n := range node.ProjectItems.Nodes {
			status := ""
			if n.FieldValueByName != nil {
				status = n.FieldValueByName.Name
			}
			found[n.Project.Number] = ProjectStatus{
				Present: true, Status: status,
				UpdatedAt: n.Project.UpdatedAt, Title: n.Project.Title,
			}
		}
		res[num] = found
	}
	return res, true
}
