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
	Estimate  int        `json:"estimate,omitempty"` // Custom field for estimate
	Status    string     `json:"status,omitempty"`   // Custom field for status
}

type Sprint struct {
	Duration    int    `json:"duration"`
	IterationId string `json:"iterationId"`
	StartDate   string `json:"startDate"`
	Title       string `json:"title"`
}

type GenericCount struct {
	TotalCount int `json:"totalCount"`
}

type ProjectDetails struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	ShortDesc string       `json:"shortDescription"`
	URL       string       `json:"url"`
	README    string       `json:"readme"`
	Number    int          `json:"number"`
	Public    bool         `json:"public"`
	Closed    bool         `json:"closed"`
	Fields    GenericCount `json:"fields"`
	Items     GenericCount `json:"items"`
	Owner     struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
}

type ProjectItemContent struct {
	Body   string `json:"body"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	URL    string `json:"url"`
}

type ProjectItem struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Content    ProjectItemContent `json:"content"`
	Estimate   int                `json:"estimate"`
	Repository string             `json:"repository"`
	Labels     []string           `json:"labels"`
	Assignees  []string           `json:"assignees"`
	Milestone  *Milestone         `json:"milestone,omitempty"`
	Sprint     *Sprint            `json:"sprint,omitempty"`
	Status     string             `json:"status"`
}

type ProjectItemsResponse struct {
	Items      []ProjectItem `json:"items"`
	TotalCount int           `json:"totalCount"`
}

type ProjectFieldsResponse struct {
	Fields     []ProjectField `json:"fields"`
	TotalCount int            `json:"totalCount"`
}

type ProjectFieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectField struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Type    string               `json:"type"`
	Options []ProjectFieldOption `json:"options,omitempty"`
}

type User struct {
	Login        string `json:"login"`
	ID           int    `json:"id"`
	NodeID       string `json:"node_id"`
	AvatarURL    string `json:"avatar_url"`
	HTMLURL      string `json:"html_url"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	GravatarID   string `json:"gravatar_id"`
	URL          string `json:"url"`
	Company      string `json:"company"`
	UserViewType string `json:"user_view_type"`
	SiteAdmin    bool   `json:"site_admin"`
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
		issue.Estimate = item.Estimate
		issue.Status = item.Status
		for _, label := range item.Labels {
			if label == "story" {
				issue.Typename = "Feature"
			}
			if label == "bug" {
				issue.Typename = "Bug"
			}
			if label == "~sub-task" {
				issue.Typename = "Task"
			}
			issue.Labels = append(issue.Labels, Label{
				Name: label,
			})
		}
		issues = append(issues, issue)
	}
	return issues
}
