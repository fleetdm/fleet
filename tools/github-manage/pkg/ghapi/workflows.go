package ghapi

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

type Action struct {
	Type          ActionType `json:"type"`
	Issue         Issue      `json:"issue"`
	Project       int        `json:"project,omitempty"`        // Project ID for project-related actions
	Label         string     `json:"label,omitempty"`          // Label for label-related actions
	Status        string     `json:"status,omitempty"`         // Status for status-related actions
	Sprint        string     `json:"sprint,omitempty"`         // Sprint for sprint-related actions
	SourceProject int        `json:"source_project,omitempty"` // Source project for moving issues
}

type Status struct {
	Index int    `json:"index"`
	State string `json:"state"`
}

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
				// log.Printf("Error adding label to issue #%d: %v", action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Added label '%s' to issue #%d", action.Label, action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}

		case ATRemoveLabel:
			err := RemoveLabelFromIssue(action.Issue.Number, action.Label)
			if err != nil {
				// log.Printf("Error removing label from issue #%d: %v", action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Removed label '%s' from issue #%d", action.Label, action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}

		case ATAddIssueToProject:
			err := AddIssueToProject(action.Issue.Number, action.Project)
			if err != nil {
				// log.Printf("Error adding issue #%d to project %d: %v", action.Issue.Number, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Added issue #%d to project %d", action.Issue.Number, action.Project)
			statusChan <- Status{Index: i, State: "success"}
		case ATRemoveIssueFromProject:
			err := RemoveIssueFromProject(action.Issue.Number, action.Project)
			if err != nil {
				// log.Printf("Error removing issue #%d from project %d: %v", action.Issue.Number, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Removed issue #%d from project %d", action.Issue.Number, action.Project)
			statusChan <- Status{Index: i, State: "success"}

		case ATSetStatus:
			err := SetIssueStatus(action.Issue.Number, action.Project, action.Status)
			if err != nil {
				// log.Printf("Error setting status for issue #%d: %v", action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Set status '%s' for issue #%d", action.Status, action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}

		case ATSyncEstimate:
			err := SyncEstimateField(action.Issue.Number, action.SourceProject, action.Project)
			if err != nil {
				// log.Printf("Error syncing estimate for issue #%d: %v", action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Synced estimate for issue #%d", action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}

		case ATSetSprint:
			err := SetCurrentSprint(action.Issue.Number, action.Project)
			if err != nil {
				// log.Printf("Error setting sprint for issue #%d: %v", action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			// log.Printf("Set sprint for issue #%d", action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}
		default:
			// log.Printf("Unknown action type: %s", action.Type)
			statusChan <- Status{Index: i, State: "error"}
		}
	}
}

func BulkAddLabel(issues []Issue, label string) error {
	for _, issue := range issues {
		err := AddLabelToIssue(issue.Number, label)
		if err != nil {
			// log.Printf("Error adding label '%s' to issue #%d: %v", label, issue.Number, err)
			return err
		}
	}
	return nil
}

func BulkRemoveLabel(issues []Issue, label string) error {
	for _, issue := range issues {
		err := RemoveLabelFromIssue(issue.Number, label)
		if err != nil {
			// log.Printf("Error removing label '%s' from issue #%d: %v", label, issue.Number, err)
			return err
		}
	}
	return nil
}

func BulkSprintKickoff(issues []Issue, sourceProjectID, projectID int) error {
	// Add ticket to the target product group project
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, projectID)
		if err != nil {
			// log.Printf("Error adding issue #%d to project %d: %v", issue.Number, projectID, err)
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
			// log.Printf("Error syncing estimate for issue #%d: %v", issue.Number, err)
			return err
		}
	}

	// Set the sprint to the current sprint
	for _, issue := range issues {
		err := SetCurrentSprint(issue.Number, projectID)
		if err != nil {
			// log.Printf("Error setting sprint for issue #%d: %v", issue.Number, err)
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
			// log.Printf("Error removing issue #%d from drafting project: %v", issue.Number, err)
			return err
		}
	}

	return nil
}

func BulkMilestoneClose(issues []Issue) error {
	// Add ticket to the drafting project
	draftingProjectID := Aliases["draft"]
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, draftingProjectID)
		if err != nil {
			// log.Printf("Error adding issue #%d to drafting project: %v", issue.Number, err)
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
			// log.Printf("Error setting status for issue #%d: %v", issue.Number, err)
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
