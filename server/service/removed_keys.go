package service

import (
	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

func init() {
	endpointer.SetRemovedKeyMessage(removedKeyMessage)
	endpointer.SetDeletedKeyRules(deletedKeyRules())
	endpointer.SetRemovedNames(requestNames(removednames.KindJSONKey), requestNames(removednames.KindQueryParam),
		requestNames(removednames.KindSortKey), []string{"/osquery/", "/orbit/", "/fleetd/", "/device/"})
}

// requestNames lists the removed names of a kind that a request can no longer send.
func requestNames(kind removednames.Kind) []string {
	var names []string
	for _, entry := range removednames.Filter(kind, removednames.ScopeRequest) {
		names = append(names, entry.Old)
	}
	return names
}

// removedKeyMessage words a removed request key with its pkg/removednames entry.
func removedKeyMessage(key string) string {
	entry, ok := removednames.Find(removednames.KindJSONKey, key, "")
	if !ok {
		// A differently cased or unregistered name gets the same wording; the registration tests catch a missing entry.
		entry = removednames.Entry{Kind: removednames.KindJSONKey, Old: key}
	}
	return entry.Message()
}

// deletedKeyRules rejects request keys that were removed without a replacement.
func deletedKeyRules() []endpointer.RemovedKeyRule {
	var rules []endpointer.RemovedKeyRule
	for _, entry := range removednames.Filter(removednames.KindJSONKey, removednames.ScopeRequest) {
		if entry.New != "" {
			continue
		}
		rules = append(rules, endpointer.RemovedKeyRule{Key: entry.Old})
	}
	return rules
}
