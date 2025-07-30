package ghapi

type Author struct {
	Login string `json:"login"`
	IsBot bool   `json:"is_bot"`
	Name  string `json:"name"`
	ID    string `json:"id"`
}

type Label struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type Milestone struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueOn       string `json:"dueOn"`
}

type Issue struct {
	Typename  string     `json:"__typename"`
	ID        string     `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Author    Author     `json:"author"`
	Assignees []Author   `json:"assignees"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
	State     string     `json:"state"`
	Labels    []Label    `json:"labels"`
	Milestone *Milestone `json:"milestone,omitempty"`
}

type Sprint struct {
	Duration    int    `json:"duration"`
	IterationId string `json:"iterationId"`
	StartDate   string `json:"startDate"`
	Title       string `json:"title"`
}

type ProjectContent struct {
	Body   string `json:"body"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	URL    string `json:"url"`
}

type ProjectItem struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Content    ProjectContent `json:"content"`
	Estimate   int            `json:"estimate"`
	Repository string         `json:"repository"`
	Labels     []string       `json:"labels"`
	Assignees  []string       `json:"assignees"`
	Milestone  *Milestone     `json:"milestone,omitempty"`
	Sprint     *Sprint        `json:"sprint,omitempty"`
	Status     string         `json:"status"`
}

type ProjectItemsResponse struct {
	Items      []ProjectItem `json:"items"`
	TotalCount int           `json:"totalCount"`
}

func ConvertItemsToIssues(items []ProjectItem) []Issue {
	var issues []Issue
	for _, item := range items {
		issue := Issue{
			ID:     item.ID,
			Number: item.Content.Number,
			Title:  item.Content.Title,
			Body:   item.Content.Body,
		}
		if item.Milestone != nil {
			issue.Milestone = &Milestone{
				Number:      item.Milestone.Number,
				Title:       item.Milestone.Title,
				Description: item.Milestone.Description,
				DueOn:       item.Milestone.DueOn,
			}
		}
		for _, assignee := range item.Assignees {
			issue.Assignees = append(issue.Assignees, Author{
				Login: assignee,
			})
		}
		issues = append(issues, issue)
	}
	return issues
}
