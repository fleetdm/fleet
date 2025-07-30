package ghapi

import (
	"encoding/json"
	"fmt"
	"log"
)

func ParseJSONtoProjectItems(jsonData []byte, limit int) ([]ProjectItem, error) {
	var items ProjectItemsResponse
	err := json.Unmarshal(jsonData, &items)
	if err != nil {
		return nil, err
	}
	if limit > 0 && items.TotalCount > limit {
		fmt.Printf("Warning: The number of items returned exceeds the specified limit. Only the first ", limit, " items will be returned.\n")
	}
	return items.Items, nil
}

func GetProjectItems(projectID, limit int) ([]ProjectItem, error) {
	// Run the command to get project items
	results, err := RunCommandAndParseJSON(fmt.Sprintf("gh project item-list --owner fleetdm --format json --limit %d %d", limit, projectID))
	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		return nil, err
	}
	items, err := ParseJSONtoProjectItems(results, limit)
	if err != nil {
		log.Printf("Error parsing issues: %v", err)
		return nil, err
	}
	log.Printf("Fetched %d items from project %d", len(items), projectID)
	return items, nil
}
