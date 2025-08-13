package ghapi

import "fleetdm/gm/pkg/logger"

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

func CreateBulkSprintKickoffActions(issues []Issue, sourceProjectID, projectID int) []Action {
	logger.Infof("Creating sprint kickoff actions for %d issues (source project: %d, target project: %d)", len(issues), sourceProjectID, projectID)

	actions := CreateBulkAddIssueToProjectAction(issues, projectID)
	actions = append(actions, CreateBulkAddLableAction(issues, ":release")...)
	actions = append(actions, CreateBulkSyncEstimateAction(issues, sourceProjectID, projectID)...)
	actions = append(actions, CreateBulkSetSprintAction(issues, projectID)...)
	actions = append(actions, CreateBulkRemoveLabelAction(issues, ":product")...)
	actions = append(actions, CreateBulkRemoveIssueFromProjectAction(issues, Aliases["draft"])...)

	logger.Infof("Created %d sprint kickoff actions", len(actions))
	return actions
}

func CreateBulkMilestoneCloseActions(issues []Issue) []Action {
	logger.Infof("Creating milestone close actions for %d issues", len(issues))

	actions := CreateBulkAddIssueToProjectAction(issues, Aliases["draft"])
	actions = append(actions, CreateBulkAddLableAction(issues, ":product")...)
	actions = append(actions, CreateBulkSetStatusAction(issues, Aliases["draft"], "confirm and")...)
	actions = append(actions, CreateBulkRemoveLabelAction(issues, ":release")...)

	logger.Infof("Created %d milestone close actions", len(actions))
	return actions
}

func CreateBulkKickOutOfSprintActions(issues []Issue, sourceProjectID int) []Action {
	logger.Infof("Creating kick out of sprint actions for %d issues (source project: %d)", len(issues), sourceProjectID)

	actions := CreateBulkAddIssueToProjectAction(issues, Aliases["draft"])
	actions = append(actions, CreateBulkSetStatusAction(issues, Aliases["draft"], "estimated")...)
	actions = append(actions, CreateBulkSyncEstimateAction(issues, sourceProjectID, Aliases["draft"])...)
	actions = append(actions, CreateBulkAddLableAction(issues, ":product")...)
	actions = append(actions, CreateBulkRemoveLabelAction(issues, ":release")...)
	actions = append(actions, CreateBulkRemoveIssueFromProjectAction(issues, sourceProjectID)...)

	logger.Infof("Created %d kick out of sprint actions", len(actions))
	return actions
}

// AsyncManager takes a list of actions and a channel to process them assynchronously.
// This will allow to send status back on the channel for live updates. the channel must return index of the action
// and the status of the action.
func AsyncManager(actions []Action, statusChan chan<- Status) {
	defer close(statusChan)

	logger.Infof("Starting AsyncManager with %d actions", len(actions))

	for i, action := range actions {
		logger.Infof("Processing action %d/%d: %s for issue #%d", i+1, len(actions), action.Type, action.Issue.Number)

		switch action.Type {
		case ATAddLabel:
			err := AddLabelToIssue(action.Issue.Number, action.Label)
			if err != nil {
				logger.Errorf("Failed to add label '%s' to issue #%d: %v", action.Label, action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully added label '%s' to issue #%d", action.Label, action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}

		case ATRemoveLabel:
			err := RemoveLabelFromIssue(action.Issue.Number, action.Label)
			if err != nil {
				logger.Errorf("Failed to remove label '%s' from issue #%d: %v", action.Label, action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully removed label '%s' from issue #%d", action.Label, action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}

		case ATAddIssueToProject:
			err := AddIssueToProject(action.Issue.Number, action.Project)
			if err != nil {
				logger.Errorf("Failed to add issue #%d to project %d: %v", action.Issue.Number, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully added issue #%d to project %d", action.Issue.Number, action.Project)
			statusChan <- Status{Index: i, State: "success"}
		case ATRemoveIssueFromProject:
			err := RemoveIssueFromProject(action.Issue.Number, action.Project)
			if err != nil {
				logger.Errorf("Failed to remove issue #%d from project %d: %v", action.Issue.Number, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully removed issue #%d from project %d", action.Issue.Number, action.Project)
			statusChan <- Status{Index: i, State: "success"}

		case ATSetStatus:
			err := SetIssueStatus(action.Issue.Number, action.Project, action.Status)
			if err != nil {
				logger.Errorf("Failed to set status '%s' for issue #%d in project %d: %v", action.Status, action.Issue.Number, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully set status '%s' for issue #%d in project %d", action.Status, action.Issue.Number, action.Project)
			statusChan <- Status{Index: i, State: "success"}

		case ATSyncEstimate:
			err := SyncEstimateField(action.Issue.Number, action.SourceProject, action.Project)
			if err != nil {
				logger.Errorf("Failed to sync estimate for issue #%d from project %d to project %d: %v", action.Issue.Number, action.SourceProject, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully synced estimate for issue #%d from project %d to project %d", action.Issue.Number, action.SourceProject, action.Project)
			statusChan <- Status{Index: i, State: "success"}

		case ATSetSprint:
			err := SetCurrentSprint(action.Issue.Number, action.Project)
			if err != nil {
				logger.Errorf("Failed to set current sprint for issue #%d in project %d: %v", action.Issue.Number, action.Project, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully set current sprint for issue #%d in project %d", action.Issue.Number, action.Project)
			statusChan <- Status{Index: i, State: "success"}
		default:
			logger.Errorf("Unknown action type: %s for issue #%d", action.Type, action.Issue.Number)
			statusChan <- Status{Index: i, State: "error"}
		}
	}

	logger.Info("AsyncManager completed all actions")
}

// BulkAddLabel adds a label to multiple issues.
func BulkAddLabel(issues []Issue, label string) error {
	logger.Infof("Adding label '%s' to %d issues", label, len(issues))
	for _, issue := range issues {
		err := AddLabelToIssue(issue.Number, label)
		if err != nil {
			logger.Errorf("Failed to add label '%s' to issue #%d: %v", label, issue.Number, err)
			return err
		}
		logger.Debugf("Added label '%s' to issue #%d", label, issue.Number)
	}
	logger.Infof("Successfully added label '%s' to all %d issues", label, len(issues))
	return nil
}

// BulkRemoveLabel removes a label from multiple issues.
func BulkRemoveLabel(issues []Issue, label string) error {
	logger.Infof("Removing label '%s' from %d issues", label, len(issues))
	for _, issue := range issues {
		err := RemoveLabelFromIssue(issue.Number, label)
		if err != nil {
			logger.Errorf("Failed to remove label '%s' from issue #%d: %v", label, issue.Number, err)
			return err
		}
		logger.Debugf("Removed label '%s' from issue #%d", label, issue.Number)
	}
	logger.Infof("Successfully removed label '%s' from all %d issues", label, len(issues))
	return nil
}

// BulkSprintKickoff performs the sprint kickoff workflow for multiple issues.
// This includes adding issues to the target project, adding release labels, syncing estimates, and setting sprints.
func BulkSprintKickoff(issues []Issue, sourceProjectID, projectID int) error {
	logger.Infof("Starting sprint kickoff workflow for %d issues (source: %d, target: %d)", len(issues), sourceProjectID, projectID)

	// Add ticket to the target product group project
	logger.Info("Step 1/6: Adding issues to target project")
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, projectID)
		if err != nil {
			logger.Errorf("Failed to add issue #%d to project %d: %v", issue.Number, projectID, err)
			return err
		}
		logger.Debugf("Added issue #%d to project %d", issue.Number, projectID)
	}

	// Add the `:release` label to each issue
	logger.Info("Step 2/6: Adding release labels")
	err := BulkAddLabel(issues, ":release")
	if err != nil {
		return err
	}

	// Sync the Estimate field from drafting project to the target product group project
	logger.Info("Step 3/6: Syncing estimates")
	for _, issue := range issues {
		err := SyncEstimateField(issue.Number, sourceProjectID, projectID)
		if err != nil {
			logger.Errorf("Failed to sync estimate for issue #%d: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Synced estimate for issue #%d", issue.Number)
	}

	// Set the sprint to the current sprint
	logger.Info("Step 4/6: Setting current sprint")
	for _, issue := range issues {
		err := SetCurrentSprint(issue.Number, projectID)
		if err != nil {
			logger.Errorf("Failed to set current sprint for issue #%d: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Set current sprint for issue #%d", issue.Number)
	}

	// Remove the `:product` label from each issue
	logger.Info("Step 5/6: Removing product labels")
	err = BulkRemoveLabel(issues, ":product")
	if err != nil {
		return err
	}

	// Remove from the drafting project
	logger.Info("Step 6/6: Removing from drafting project")
	draftingProjectID := Aliases["draft"]
	for _, issue := range issues {
		err := RemoveIssueFromProject(issue.Number, draftingProjectID)
		if err != nil {
			logger.Errorf("Failed to remove issue #%d from drafting project: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Removed issue #%d from drafting project", issue.Number)
	}

	logger.Info("Sprint kickoff workflow completed successfully")
	return nil
}

// BulkMilestoneClose performs the milestone close workflow for multiple issues.
// This includes moving issues back to the drafting project and removing them from product group projects.
func BulkMilestoneClose(issues []Issue) error {
	logger.Infof("Starting milestone close workflow for %d issues", len(issues))

	// Add ticket to the drafting project
	logger.Info("Step 1/4: Adding issues to drafting project")
	draftingProjectID := Aliases["draft"]
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, draftingProjectID)
		if err != nil {
			logger.Errorf("Failed to add issue #%d to drafting project: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Added issue #%d to drafting project", issue.Number)
	}

	// Add the `:product` label to each issue
	logger.Info("Step 2/4: Adding product labels")
	err := BulkAddLabel(issues, ":product")
	if err != nil {
		return err
	}

	// Set the status to "confirm and celebrate"
	logger.Info("Step 3/4: Setting status to 'confirm and celebrate'")
	for _, issue := range issues {
		err := SetIssueStatus(issue.Number, draftingProjectID, "confirm and celebrate")
		if err != nil {
			logger.Errorf("Failed to set status for issue #%d: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Set status for issue #%d", issue.Number)
	}

	// Remove the `:release` label from each issue
	logger.Info("Step 4/4: Removing release labels")
	err = BulkRemoveLabel(issues, ":release")
	if err != nil {
		return err
	}

	logger.Info("Milestone close workflow completed successfully")
	return nil
}

// BulkKickOutOfSprint performs the kick out of sprint workflow for multiple issues.
// This includes moving issues back to the drafting project, setting status to estimated,
// syncing estimates from source project, and updating labels.
func BulkKickOutOfSprint(issues []Issue, sourceProjectID int) error {
	logger.Infof("Starting kick out of sprint workflow for %d issues (source: %d)", len(issues), sourceProjectID)

	// Add issues to the drafting project
	logger.Info("Step 1/5: Adding issues to drafting project")
	draftingProjectID := Aliases["draft"]
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, draftingProjectID)
		if err != nil {
			logger.Errorf("Failed to add issue #%d to drafting project: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Added issue #%d to drafting project", issue.Number)
	}

	// Set the status to "estimated"
	logger.Info("Step 2/5: Setting status to 'estimated'")
	for _, issue := range issues {
		err := SetIssueStatus(issue.Number, draftingProjectID, "estimated")
		if err != nil {
			logger.Errorf("Failed to set status for issue #%d: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Set status for issue #%d", issue.Number)
	}

	// Sync the Estimate field from source project to the drafting project
	logger.Info("Step 3/5: Syncing estimates from source project")
	for _, issue := range issues {
		err := SyncEstimateField(issue.Number, sourceProjectID, draftingProjectID)
		if err != nil {
			logger.Errorf("Failed to sync estimate for issue #%d: %v", issue.Number, err)
			return err
		}
		logger.Debugf("Synced estimate for issue #%d", issue.Number)
	}

	// Add the `:product` label to each issue
	logger.Info("Step 4/5: Adding product labels")
	err := BulkAddLabel(issues, ":product")
	if err != nil {
		return err
	}

	// Remove the `:release` label from each issue
	logger.Info("Step 5/5: Removing release labels")
	err = BulkRemoveLabel(issues, ":release")
	if err != nil {
		return err
	}

	logger.Info("Kick out of sprint workflow completed successfully")
	return nil
}
