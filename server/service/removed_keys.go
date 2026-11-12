package service

import (
	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

func init() {
	endpointer.SetRemovedKeyMessage(removedKeyMessage)
	endpointer.SetDeletedKeyRules(deletedKeyRules())
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
