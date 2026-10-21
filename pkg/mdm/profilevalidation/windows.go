package profilevalidation

import (
	"encoding/json"
	"strings"
)

// WindowsVersion pins the Microsoft DDF v2 drop the Windows section was generated from.
type WindowsVersion struct {
	Release string `json:"release"` // the zip's name, e.g. "DDFv2Feb2026"
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// WindowsTemplatesVersion pins the third-party ADMX template data the profile generator ships.
type WindowsTemplatesVersion struct {
	SHA256      string `json:"sha256"`
	GeneratedAt string `json:"generated_at"`
}

// WindowsNode is one leaf LocURI, keyed in the schema by its declared path. Dynamic segments (an
// empty NodeName in the DDF) are "*".
type WindowsNode struct {
	Format WindowsFormats `json:"format"`
	Scope  WindowsScope   `json:"scope"`

	// Allowed values; at most one kind is set, as in the DDF.
	Enum  []string `json:"enum,omitempty"`
	Range *Range   `json:"range,omitempty"`
	RegEx string   `json:"regex,omitempty"`
	Flag  []string `json:"flag,omitempty"`
	XSD   string   `json:"xsd,omitempty"` // sha256 key into windows_xsds
	// ADMXBacked nodes take an ADMX policy payload; the DDF says nothing more about their values.
	ADMXBacked bool `json:"admx_backed,omitempty"`
	// Delimiter is set when the value is a list of allowed values joined by it.
	Delimiter string `json:"delimiter,omitempty"`

	// MinBuild lists the first build per servicing branch, as declared (some are placeholders).
	MinBuild   []string           `json:"min_build,omitempty"`
	Deprecated *WindowsDeprecated `json:"deprecated,omitempty"`

	// Source is "ddf" (omitted) or "admx_template".
	Source   string `json:"source,omitempty"`
	Override string `json:"override,omitempty"`
}

// WindowsDeprecated marks a deprecated node; Build is set when the DDF names the build.
type WindowsDeprecated struct {
	Build string `json:"build,omitempty"`
}

type WindowsScope string

const (
	WindowsScopeDevice WindowsScope = "device"
	WindowsScopeUser   WindowsScope = "user"
)

const WindowsSourceADMXTemplate = "admx_template"

// WindowsFormat is a DDF DFFormat. Values outside this list are carried as declared.
type WindowsFormat string

const (
	WindowsFormatInt   WindowsFormat = "int"
	WindowsFormatChr   WindowsFormat = "chr"
	WindowsFormatBool  WindowsFormat = "bool"
	WindowsFormatB64   WindowsFormat = "b64"
	WindowsFormatXML   WindowsFormat = "xml"
	WindowsFormatNode  WindowsFormat = "node"
	WindowsFormatNull  WindowsFormat = "null"
	WindowsFormatBin   WindowsFormat = "bin"
	WindowsFormatTime  WindowsFormat = "time"
	WindowsFormatDate  WindowsFormat = "date"
	WindowsFormatFloat WindowsFormat = "float"
	// WindowsFormatAny is only written by a loosen override.
	WindowsFormatAny WindowsFormat = "any"
)

// WindowsFormats is a node's format. It's a single string in JSON unless an override widened it.
type WindowsFormats []WindowsFormat

func (f WindowsFormats) MarshalJSON() ([]byte, error) {
	if len(f) == 1 {
		return json.Marshal(f[0])
	}
	return json.Marshal([]WindowsFormat(f))
}

func (f *WindowsFormats) UnmarshalJSON(data []byte) error {
	var s WindowsFormat
	if err := json.Unmarshal(data, &s); err == nil {
		*f = WindowsFormats{s}
		return nil
	}
	var list []WindowsFormat
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*f = list
	return nil
}

// WindowsNode returns the node for a LocURI, or nil if the schema doesn't know it. A "*" segment in
// the schema matches any one non-empty segment; an exact match wins, then the match with the fewest
// "*" segments. Paths are compared as written: ./Vendor/MSFT/X and ./Device/Vendor/MSFT/X differ.
func (s *ProfileValidationSchema) WindowsNode(locURI string) *WindowsNode {
	if n := s.Windows[locURI]; n != nil {
		return n
	}
	want := strings.Split(locURI, "/")
	var best *WindowsNode
	bestKey, bestStars := "", 0
	// scans every node on a miss (~5k); we can index the "*" paths at Parse if lookups get hot.
	for key, n := range s.Windows {
		stars, ok := matchWindowsPath(strings.Split(key, "/"), want)
		if !ok || n == nil {
			continue
		}
		// Ties break on the key so the result doesn't depend on map order.
		if best == nil || stars < bestStars || (stars == bestStars && key < bestKey) {
			best, bestKey, bestStars = n, key, stars
		}
	}
	return best
}

func matchWindowsPath(pattern, path []string) (stars int, ok bool) {
	if len(pattern) != len(path) {
		return 0, false
	}
	for i, seg := range pattern {
		switch {
		case seg == "*" && path[i] != "":
			stars++
		case seg != path[i]:
			return 0, false
		}
	}
	return stars, true
}
