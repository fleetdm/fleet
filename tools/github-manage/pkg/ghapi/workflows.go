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
