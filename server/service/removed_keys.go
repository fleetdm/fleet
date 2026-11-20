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
	endpointer.SetRemovedFormFields(requestNames(removednames.KindFormField))
}

// requestNames lists the removed names of a kind that a request can no longer send. An entry with a Context belongs to one
// endpoint, which rejects it itself, so it stays out of the rules every request goes through.
func requestNames(kind removednames.Kind) []string {
	var names []string
	for _, entry := range removednames.Filter(kind, removednames.ScopeRequest) {
		if entry.Context != "" {
			continue
		}
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
		if entry.New != "" || entry.Context != "" {
			continue
		}
		rules = append(rules, endpointer.RemovedKeyRule{Key: entry.Old})
	}
	return rules
}
