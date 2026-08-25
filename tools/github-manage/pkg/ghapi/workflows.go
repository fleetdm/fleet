package ghapi

import (
	"strings"

	"fleetdm/gm/pkg/logger"
)

// ActionType represents the type of action to be performed on an issue.
type ActionType string

// newworkflow new actions added must have a new actiontype
const (
	ATAddLabel               ActionType = "add_label"
	ATRemoveLabel            ActionType = "remove_label"
	ATAddIssueToProject      ActionType = "add_issue_to_project"
	ATRemoveIssueFromProject ActionType = "remove_issue_from_project"
	ATSetStatus              ActionType = "set_status"
	ATSyncEstimate           ActionType = "sync_estimate"
	ATSetSprint              ActionType = "set_sprint"
	ATCloseIssue             ActionType = "close_issue"
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

// newworkflow new workflow 'action builder' functions are here
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

// CreateBulkMoveToCurrentSprintIfNotReadyQA creates actions to set current sprint for issues whose
// status does NOT contain "ready" or "qa" (case-insensitive) in the given project.
func CreateBulkMoveToCurrentSprintIfNotReadyQA(issues []Issue, projectID int) []Action {
	var filtered []Issue
	for _, is := range issues {
		s := strings.ToLower(strings.TrimSpace(is.Status))
		if s == "" {
			// No status -> include (move to current sprint)
			filtered = append(filtered, is)
			continue
		}
		if strings.Contains(s, "ready") || strings.Contains(s, "qa") {
			// Skip statuses with ready or qa
			continue
		}
		filtered = append(filtered, is)
	}
	return CreateBulkSetSprintAction(filtered, projectID)
}

func CreateBulkMilestoneCloseActions(issues []Issue) []Action {
	logger.Infof("Creating milestone close actions for %d issues", len(issues))
	// Split issues by type
	var storyIssues []Issue
	var closeIssues []Issue
	for _, is := range issues {
		isStory := false
		isBug := false
		isSubTask := false
		for _, l := range is.Labels {
			name := strings.ToLower(strings.TrimSpace(l.Name))
			if name == "story" {
				isStory = true
			} else if name == "bug" {
				isBug = true
			} else if name == "~sub-task" {
				isSubTask = true
			}
		}
		if isStory {
			storyIssues = append(storyIssues, is)
		} else if isBug || isSubTask {
			closeIssues = append(closeIssues, is)
		} else {
			// default: treat as story to keep previous behavior
			storyIssues = append(storyIssues, is)
		}
	}

	var actions []Action
	if len(storyIssues) > 0 {
		for _, issue := range storyIssues {
			// Stories already live on their product group's board (there's no
			// separate drafting board to move them onto) — resolve which board
			// that is from the issue's own #g-* label and set status there.
			projectID, ok := ProjectIDForLabels(issue.Labels)
			if !ok {
				logger.Errorf("Skipping 'confirm and celebrate' status for issue #%d: no product group label found", issue.Number)
				continue
			}
			actions = append(actions, CreateBulkSetStatusAction([]Issue{issue}, projectID, "confirm and celebrate")...)
		}
		actions = append(actions, CreateBulkRemoveLabelAction(storyIssues, ":release")...)
	}
	for _, is := range closeIssues {
		actions = append(actions, Action{Type: ATCloseIssue, Issue: is})
	}

	logger.Infof("Created %d milestone close actions (stories=%d, close=%d)", len(actions), len(storyIssues), len(closeIssues))
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
		// newworkflow new actions must be supported in this switch
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
		case ATCloseIssue:
			err := CloseIssue(action.Issue.Number)
			if err != nil {
				logger.Errorf("Failed to close issue #%d: %v", action.Issue.Number, err)
				statusChan <- Status{Index: i, State: "error"}
				continue
			}
			logger.Infof("Successfully closed issue #%d", action.Issue.Number)
			statusChan <- Status{Index: i, State: "success"}
		default:
			logger.Errorf("Unknown action type: %s for issue #%d", action.Type, action.Issue.Number)
			statusChan <- Status{Index: i, State: "error"}
		}
	}

	logger.Info("AsyncManager completed all actions")
}

//
// IMPORTANT: TUI workflows should use the Async methods above instead of the synchronous methods below
//

// These methods are more for testing individual steps w/o tui.
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

// BulkMilestoneClose performs the milestone close workflow for multiple issues.
// Stories already live on their product group's board, so this sets status
// directly on that board (resolved per-issue from its #g-* label) instead of
// moving anything to a separate drafting project.
func BulkMilestoneClose(issues []Issue) error {
	logger.Infof("Starting milestone close workflow for %d issues", len(issues))
	var storyIssues []Issue
	var closeIssues []Issue
	for _, is := range issues {
		isStory := false
		isBug := false
		isSubTask := false
		for _, l := range is.Labels {
			name := strings.ToLower(strings.TrimSpace(l.Name))
			if name == "story" {
				isStory = true
			} else if name == "bug" {
				isBug = true
			} else if name == "~sub-task" {
				isSubTask = true
			}
		}
		if isStory {
			storyIssues = append(storyIssues, is)
		} else if isBug || isSubTask {
			closeIssues = append(closeIssues, is)
		} else {
			storyIssues = append(storyIssues, is)
		}
	}

	// Process story issues: set status to confirm and celebrate on their own
	// product group board, remove :release
	if len(storyIssues) > 0 {
		logger.Infof("Processing %d story issues for milestone close", len(storyIssues))
		// Step 1: Set status to confirm and celebrate on each issue's product group board
		for _, issue := range storyIssues {
			projectID, ok := ProjectIDForLabels(issue.Labels)
			if !ok {
				logger.Errorf("Skipping issue #%d: no product group label found, cannot resolve its board", issue.Number)
				continue
			}
			err := SetIssueStatus(issue.Number, projectID, "confirm and celebrate")
			if err != nil {
				logger.Errorf("Failed to set status for issue #%d: %v", issue.Number, err)
				return err
			}
		}
		// Step 2: Remove :release
		if err := BulkRemoveLabel(storyIssues, ":release"); err != nil {
			return err
		}
	}

	// Process bug/sub-task issues: close them
	if len(closeIssues) > 0 {
		logger.Infof("Closing %d bug/sub-task issues for milestone close", len(closeIssues))
		for _, issue := range closeIssues {
			if err := CloseIssue(issue.Number); err != nil {
				logger.Errorf("Failed to close issue #%d: %v", issue.Number, err)
				return err
			}
		}
	}

	logger.Info("Milestone close workflow completed successfully")
	return nil
}
