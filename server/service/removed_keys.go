package service

import (
	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

func init() {
	endpointer.SetRemovedKeyMessage(removedKeyMessage)
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
