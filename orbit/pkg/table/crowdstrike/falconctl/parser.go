// based on github.com/kolide/launcher/pkg/osquery/tables
package falconctl

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// parseOptions parses the stdout returned from falconctl's displayed options. As far as we know, output is a single
// line, comma-separated. We parse multiple lines, but assume data does not space that. Eg: linebreaks and commas
// treated as seperators.
func parseOptions(reader io.Reader) (any, error) {
	results := make(map[string]interface{})
	errors := make([]error, 0)

	// rfm-reason, oddly, produces two KV pairs on a single line. We need to track the last key we saw, and
	// append to that value.
	var lastKey string

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// sometimes lines end in , or ., remove them.
		line = strings.TrimRight(line, ",.")
		if line == "" {
			continue
		}

		pairs := strings.Split(line, ", ")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)

			// The format is quite inconsistent. The following sample shows 4 possible
			// outputs. We'll try to parse them all:
			//
			//	cid="ac917ab****************************"
			//	aid is not set
			//	aph is not set
			//	app is not set
			//	rfm-state is not set
			//	rfm-reason is not set
			//  rfm-reason=None, code=0x0,
			//	feature is not set
			//	metadata-query=enable (unset default)
			//	version = 6.38.13501.0
			// We see 4 different formats. We'll try to parse them all.

			if strings.HasSuffix(pair, " is not set") {
				// What should this be set to? nil? "is not set"? TBD!
				results[pair[:len(pair)-len(" is not set")]] = "is not set"
				continue
			}

			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				// remove quotes and extra spaces
				kv[0] = strings.Trim(kv[0], `" `)
				kv[1] = strings.Trim(kv[1], `" `)

				// Remove parenthetical note about an unset default
				kv[1] = strings.TrimSuffix(kv[1], " (unset default)")

				if lastKey == "rfm-reason" && kv[0] == "code" {
					kv[0] = "rfm-reason-code"
				}

				if kv[0] == "tags" {
					results[kv[0]] = strings.Split(kv[1], ",")
					continue
				}

				results[kv[0]] = kv[1]
				lastKey = kv[0]
				continue
			}

			// Unknown format. Note the error
			errors = append(errors, fmt.Errorf("unknown format: `%s` on line `%s`", pair, line))
		}

	}

	if len(errors) > 0 {
		return results, fmt.Errorf("errors parsing: %v", errors)
	}
	return results, nil
}
