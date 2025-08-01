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
	for _, issue := range issues {
		err := AddIssueToProject(issue.Number, projectID)
		if err != nil {
			log.Printf("Error adding issue #%d to project %d: %v", issue.Number, projectID, err)
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
		err := SyncEstimateField(issue.Number, projectID)
		if err != nil {
			log.Printf("Error syncing estimate for issue #%d: %v", issue.Number, err)
			return err
		}
	}

	// Set the sprint to the current sprint
	for _, issue := range issues {
		err := SetCurrentSprint(issue.Number, projectID)
		if err != nil {
			log.Printf("Error setting sprint for issue #%d: %v", issue.Number, err)
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
			log.Printf("Error removing issue #%d from drafting project: %v", issue.Number, err)
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
			log.Printf("Error adding issue #%d to drafting project: %v", issue.Number, err)
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
			log.Printf("Error setting status for issue #%d: %v", issue.Number, err)
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
