package main

import (
	"fleetdm/gm/pkg/ghapi"
	"reflect"
	"testing"
)

func TestFindIssuesWithHistoricalLabel(t *testing.T) {
	labelName := "bug"

	issues := []ghapi.Issue{
		{
			Number: 1,
			Labels: []ghapi.Label{{Name: "bug"}},
		},
		{
			Number: 2,
			Labels: []ghapi.Label{{Name: "feature"}},
		},
		{
			Number: 3,
			Labels: []ghapi.Label{},
		},
	}

	timelineMap := map[int][]ghapi.TimelineEvent{
		2: {
			{Event: "labeled", Label: ghapi.Label{Name: "bug"}},
			{Event: "unlabeled", Label: ghapi.Label{Name: "bug"}},
		},
		3: {
			{Event: "labeled", Label: ghapi.Label{Name: "enhancement"}},
		},
	}

	timelineFetcher := func(num int) ([]ghapi.TimelineEvent, error) {
		return timelineMap[num], nil
	}

	expected := []int{1, 2}
	result, err := findIssuesWithHistoricalLabel(issues, labelName, timelineFetcher)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
