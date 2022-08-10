package fleet

import (
	"fmt"
	"regexp"
	"strconv"
)

type WindowsUpdate struct {
	KBID      uint `db:"kb_id"`
	DateEpoch uint `db:"date_epoch"`
}

// NewWindowsUpdate returns a new WindowsUpdate from the provided props:
// - title: The title of the windows update (see
// https://osquery.io/schema/5.4.0/#windows_update_history)
// - dateEpoch: The date the update was applied on (see
// https://osquery.io/schema/5.4.0/#windows_update_history)
func NewWindowsUpdate(title string, dateEpoch string) (WindowsUpdate, error) {
	kbID, err := parseKBID(title)
	if err != nil {
		return WindowsUpdate{}, err
	}

	dEpoch, err := parseDateEpoch(dateEpoch)
	if err != nil {
		return WindowsUpdate{}, err
	}

	return WindowsUpdate{
		KBID:      kbID,
		DateEpoch: dEpoch,
	}, nil
}

func (wu WindowsUpdate) MoreRecent(other WindowsUpdate) bool {
	return wu.DateEpoch > other.DateEpoch
}

func parseDateEpoch(val string) (uint, error) {
	dEpoch, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}

	if dEpoch < 0 {
		return 0, fmt.Errorf("invalid epoch value %d", dEpoch)
	}

	return uint(dEpoch), nil
}

// parseKBID extracts the KB (Knowledge Base) id contained inside a string. KB ids are found based on
// the pattern 'KB\d+'. In case of multiple matches, the id
// will be based on the last match. Will return an error if:
// - No matches are found
// - The matched KB contains an 'invalid' id (< 0)
func parseKBID(str string) (uint, error) {
	r := regexp.MustCompile(`\s?\(?KB(?P<Id>\d+)\s?\)?`)
	m := r.FindAllStringSubmatch(str, -1)
	idx := r.SubexpIndex("Id")

	if len(m) == 0 || idx <= 0 {
		return 0, fmt.Errorf("KB id not found in %s", str)
	}

	last := m[len(m)-1]
	id, err := strconv.Atoi(last[idx])
	if err != nil {
		return 0, err
	}

	if id <= 0 {
		return 0, fmt.Errorf("Invalid KB id value found in %s", str)
	}

	return uint(id), nil
}
