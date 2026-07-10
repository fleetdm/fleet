package jarvis

import (
	"fmt"

	"fleetdm/gm/pkg/ghapi"
)

// Status intents jarvis writes. Each is an ordered list of candidate substrings
// matched case-insensitively against the board's actual Status options (which
// carry emoji/wording variations), so the first that exists on the board wins.
// Fleet boards use "🐥 Ready for review" rather than "In review", so that's tried
// first; "in review" is a fallback for boards that name it that way.
var (
	statusInProgress = []string{"in progress"}
	statusInReview   = []string{"ready for review", "in review"}
	statusAwaitingQA = []string{"awaiting qa"}
)

// resolveAndSetStatus sets an issue's project Status to the first board option
// matching one of the candidate intents, returning the resolved option name
// actually written. No-op returning "" when project is 0 (issue isn't on a board).
func resolveAndSetStatus(issue, project int, intents []string) (string, error) {
	if project == 0 {
		return "", fmt.Errorf("no project board known for issue #%d", issue)
	}
	var option string
	for _, intent := range intents {
		if opt, err := ghapi.FindFieldValueByName(project, "Status", intent); err == nil {
			option = opt
			break
		}
	}
	if option == "" {
		return "", fmt.Errorf("project %d has no Status column matching any of %v", project, intents)
	}
	if err := ghapi.SetIssueStatus(issue, project, option); err != nil {
		return "", err
	}
	return option, nil
}
