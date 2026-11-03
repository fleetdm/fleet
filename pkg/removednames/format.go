package removednames

import (
	"fmt"
	"strings"
)

// Format builds the error message for a removed name. Every message links to the migration guide.
type Format interface {
	Message(e Entry) string
}

// ReplacedFormat is the default format: it names the replacement, or just the removal when the entry has no replacement.
//
//	team_id was removed in Fleet 5. Use fleet_id instead. See <guide>
//	team_id was removed in Fleet 5. See <guide>
type ReplacedFormat struct{}

// Message implements Format.
func (ReplacedFormat) Message(e Entry) string {
	return Message(displayName(e.Kind, e.Old), displayName(e.Kind, e.New))
}

// EndpointFormat is for a removed endpoint (KindPath). It shows the HTTP method from the entry's Context, when set, and leaves the
// replacement to the guide.
//
//	The "POST /mdm/apple/profiles/match" endpoint is not available anymore. What to use instead: <guide>
type EndpointFormat struct {
	// ByDefault is for an endpoint that a setting can turn back on.
	ByDefault bool
}

// Message implements Format.
func (f EndpointFormat) Message(e Entry) string {
	endpoint := strings.TrimPrefix(e.Old, pathPrefix)
	if e.Context != "" {
		endpoint = e.Context + " " + endpoint
	}
	byDefault := ""
	if f.ByDefault {
		byDefault = " by default"
	}
	return fmt.Sprintf(`The "%s" endpoint is not available anymore%s. What to use instead: %s`, endpoint, byDefault, GuideURL)
}

// displayName is how a name appears in a message. Only flags are displayed differently: they get their leading dashes, one for a
// single-letter flag such as -t and two for the rest.
func displayName(kind Kind, name string) string {
	switch {
	case kind != KindFlag || name == "":
		return name
	case len(name) == 1:
		return "-" + name
	default:
		return "--" + name
	}
}
