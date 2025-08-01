package ghapi

// ActionType represents the type of action to be performed on an issue.
type ActionType string

const (
	ATAddLabel               ActionType = "add_label"
	ATRemoveLabel            ActionType = "remove_label"
	ATAddIssueToProject      ActionType = "add_issue_to_project"
	ATRemoveIssueFromProject ActionType = "remove_issue_from_project"
	ATSetStatus              ActionType = "set_status"
	ATSyncEstimate           ActionType = "sync_estimate"
	ATSetSprint              ActionType = "set_sprint"
)

// Action represents a single action to be performed on an issue.
type Action struct {
	Type          ActionType `json:"type"`
	Issue         Issue      `json:"issue"`
	Project       int        `json:"project,omitempty"`        // Project ID for project-related actions
	Label         string     `json:"label,omitempty"`          // Label for label-related actions
	Status        string     `json:"status,omitempty"`         // Status for status-related actions
	Sprint        string     `json:"sprint,omitempty"`         // Sprint for sprint-related actions
	SourceProject int        `json:"source_project,omitempty"` // Source project for moving issues
}

// Status represents the status of an item in a project.
type Status struct {
	Index int    `json:"index"`
	State string `json:"state"`
}

// CreateBulkAddLableAction creates actions to add a label to multiple issues.
func CreateBulkAddLableAction(issues []Issue, label string) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:  ATAddLabel,
			Issue: issue,
			Label: label,
		})
	}
	return actions
}

// CreateBulkRemoveLabelAction creates actions to remove a label from multiple issues.
func CreateBulkRemoveLabelAction(issues []Issue, label string) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:  ATRemoveLabel,
			Issue: issue,
			Label: label,
		})
	}
	return actions
}

// CreateBulkAddIssueToProjectAction creates actions to add multiple issues to a project.
func CreateBulkAddIssueToProjectAction(issues []Issue, projectID int) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:    ATAddIssueToProject,
			Issue:   issue,
			Project: projectID,
		})
	}
	return actions
}

// CreateBulkRemoveIssueFromProjectAction creates actions to remove multiple issues from a project.
func CreateBulkRemoveIssueFromProjectAction(issues []Issue, projectID int) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:    ATRemoveIssueFromProject,
			Issue:   issue,
			Project: projectID,
		})
	}
	return actions
}

// CreateBulkSetStatusAction creates actions to set the status for multiple issues in a project.
func CreateBulkSetStatusAction(issues []Issue, projectID int, status string) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:    ATSetStatus,
			Issue:   issue,
			Project: projectID,
			Status:  status,
		})
	}
	return actions
}

// CreateBulkSyncEstimateAction creates actions to sync estimates from source to target projects for multiple issues.
func CreateBulkSyncEstimateAction(issues []Issue, sourceProjectID, targetProjectID int) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:          ATSyncEstimate,
			Issue:         issue,
			SourceProject: sourceProjectID,
			Project:       targetProjectID,
		})
	}
	return actions
}

// CreateBulkSetSprintAction creates actions to set the sprint for multiple issues in a project.
func CreateBulkSetSprintAction(issues []Issue, projectID int) []Action {
	var actions []Action
	for _, issue := range issues {
		actions = append(actions, Action{
			Type:    ATSetSprint,
			Issue:   issue,
			Project: projectID,
		})
	}
	return actions
}

// AsyncManager takes a list of actions and a channel to process them assynchronously.
// This will allow to send status back on the channel for live updates. the channel must return index of the action
// and the status of the action.
func AsyncManager(actions []Action, statusChan chan<- Status) {
	defer close(statusChan)

	for i, action := range actions {
		switch action.Type {
		case ATAddLabel:
			err := AddLabelToIssue(action.Issue.Number, action.Label)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}

		case ATRemoveLabel:
			err := RemoveLabelFromIssue(action.Issue.Number, action.Label)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}

		case ATAddIssueToProject:
			err := AddIssueToProject(action.Issue.Number, action.Project)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}
		case ATRemoveIssueFromProject:
			err := RemoveIssueFromProject(action.Issue.Number, action.Project)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}

		case ATSetStatus:
			err := SetIssueStatus(action.Issue.Number, action.Project, action.Status)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}

		case ATSyncEstimate:
			err := SyncEstimateField(action.Issue.Number, action.SourceProject, action.Project)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}

		case ATSetSprint:
			err := SetCurrentSprint(action.Issue.Number, action.Project)
			if err != nil {
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			statusChan <- Status{Index: i, State: "success"}
		default:
			statusChan <- Status{Index: i, State: "error"}
		}
	}
}

// BulkAddLabel adds a label to multiple issues.
func BulkAddLabel(issues []Issue, label string) error {
	for _, issue := range issues {
		err := AddLabelToIssue(issue.Number, label)
		if err != nil {
			return err
		}
	}
	return nil
}

// BulkRemoveLabel removes a label from multiple issues.
func BulkRemoveLabel(issues []Issue, label string) error {
	for _, issue := range issues {
		err := RemoveLabelFromIssue(issue.Number, label)
		if err != nil {
			return err
		}
	}
	return nil
}

// BulkSprintKickoff performs the sprint kickoff workflow for multiple issues.
// This includes adding issues to the target project, adding release labels, syncing estimates, and setting sprints.
func BulkSprintKickoff(issues []Issue, sourceProjectID, projectID int) error {
	// Add ticket to the target product group project
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, projectID)
		if err != nil {
			return err
		}
	}

	// Add the `:release` label to each issue
	err := BulkAddLabel(issues, ":release")
	if err != nil {
		return err
	}

	// Sync the Estimate field from drafting project to the target product group project
	for _, issue := range issues {
		err := SyncEstimateField(issue.Number, sourceProjectID, projectID)
		if err != nil {
			return err
		}
	}

	// Set the sprint to the current sprint
	for _, issue := range issues {
		err := SetCurrentSprint(issue.Number, projectID)
		if err != nil {
			return err
		}
	}

	// Remove the `:product` label from each issue
	err = BulkRemoveLabel(issues, ":product")
	if err != nil {
		return err
	}

	// Remove from the drafting project
	draftingProjectID := Aliases["draft"]
	for _, issue := range issues {
		err := RemoveIssueFromProject(issue.Number, draftingProjectID)
		if err != nil {
			return err
		}
	}

	return nil
}

// BulkMilestoneClose performs the milestone close workflow for multiple issues.
// This includes moving issues back to the drafting project and removing them from product group projects.
func BulkMilestoneClose(issues []Issue) error {
	// Add ticket to the drafting project
	draftingProjectID := Aliases["draft"]
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, draftingProjectID)
		if err != nil {
			return err
		}
	}

	// Add the `:product` label to each issue
	err := BulkAddLabel(issues, ":product")
	if err != nil {
		return err
	}

	// Set the status to "confirm and celebrate"
	for _, issue := range issues {
		err := SetIssueStatus(issue.Number, draftingProjectID, "confirm and celebrate")
		if err != nil {
			return err
		}
	}

	// Remove the `:release` label from each issue
	err = BulkRemoveLabel(issues, ":release")
	if err != nil {
		return err
	}

	return nil
}
