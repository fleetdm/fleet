package ghapi

import "log"

func BulkAddLabel(issues []Issue, label string) error {
	for _, issue := range issues {
		err := AddLabelToIssue(issue.Number, label)
		if err != nil {
			log.Printf("Error adding label '%s' to issue #%d: %v", label, issue.Number, err)
			return err
		}
	}
	return nil
}

func BulkRemoveLabel(issues []Issue, label string) error {
	for _, issue := range issues {
		err := RemoveLabelFromIssue(issue.Number, label)
		if err != nil {
			log.Printf("Error removing label '%s' from issue #%d: %v", label, issue.Number, err)
			return err
		}
	}
	return nil
}

func BulkSprintKickoff(issues []Issue, projectID int) error {
	// Add ticket to the target product group project

	// Add the `:release` label to each issue

	// Sync the Estimate field from drafting project to the target product group project

	// Set the sprint to the current sprint

	// Remove the `:product` label from each issue

	// Remove from the drafting project

	return nil
}

func BulkMilestoneClose(issues []Issue) error {
	// Add ticket to the drafting project

	// Add the `:product` label to each issue

	// Set the status to "confirm and celebrate"

	// Remove the `:release` label from each issue

	return nil
}
