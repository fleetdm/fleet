package ghapi

import (
	"fmt"
	"log"
	"sync"
)

// Cache structures for performance optimization
var (
	// Cache for project number -> project node ID
	projectNodeIDCache = make(map[int]string)
	projectNodeIDMutex sync.RWMutex

	// Cache for (issue number, project ID) -> project item ID
	projectItemIDCache = make(map[string]string)
	projectItemIDMutex sync.RWMutex

	// Existing field cache
	MapProjectFieldNameToField = map[int]map[string]ProjectField{}
)

// generateProjectItemCacheKey creates a unique key for project item cache
func generateProjectItemCacheKey(issueNumber, projectID int) string {
	return fmt.Sprintf("%d:%d", issueNumber, projectID)
}

// Cache management functions

// ClearProjectNodeIDCache clears the project node ID cache
func ClearProjectNodeIDCache() {
	projectNodeIDMutex.Lock()
	defer projectNodeIDMutex.Unlock()
	projectNodeIDCache = make(map[int]string)
	log.Printf("Cleared project node ID cache")
}

// ClearProjectItemIDCache clears the project item ID cache
func ClearProjectItemIDCache() {
	projectItemIDMutex.Lock()
	defer projectItemIDMutex.Unlock()
	projectItemIDCache = make(map[string]string)
	log.Printf("Cleared project item ID cache")
}

// ClearProjectFieldsCache clears the project fields cache
func ClearProjectFieldsCache() {
	MapProjectFieldNameToField = make(map[int]map[string]ProjectField)
	log.Printf("Cleared project fields cache")
}

// ClearAllCaches clears all caches
func ClearAllCaches() {
	ClearProjectNodeIDCache()
	ClearProjectItemIDCache()
	ClearProjectFieldsCache()
	log.Printf("Cleared all caches")
}

// GetCacheStats returns statistics about cache usage
func GetCacheStats() map[string]interface{} {
	projectNodeIDMutex.RLock()
	projectNodeIDCount := len(projectNodeIDCache)
	projectNodeIDMutex.RUnlock()

	projectItemIDMutex.RLock()
	projectItemIDCount := len(projectItemIDCache)
	projectItemIDMutex.RUnlock()

	fieldCacheCount := len(MapProjectFieldNameToField)

	return map[string]interface{}{
		"project_node_ids": projectNodeIDCount,
		"project_item_ids": projectItemIDCount,
		"project_fields":   fieldCacheCount,
	}
}

// InvalidateProjectItemID removes a specific project item ID from cache
// Useful when an item might have been moved/changed
func InvalidateProjectItemID(issueNumber, projectID int) {
	cacheKey := generateProjectItemCacheKey(issueNumber, projectID)
	projectItemIDMutex.Lock()
	defer projectItemIDMutex.Unlock()
	delete(projectItemIDCache, cacheKey)
	log.Printf("Invalidated cache for issue #%d in project %d", issueNumber, projectID)
}
