// based on github.com/kolide/launcher/pkg/osquery/tables
package firmwarepasswd

import (
	"bufio"
	"bytes"

	"github.com/rs/zerolog"
)

type Matcher struct {
	Match   func(string) bool
	KeyFunc func(string) (string, error)
	ValFunc func(string) (string, error)
}

type OutputParser struct {
	matchers []Matcher
	logger   zerolog.Logger
}

func NewParser(logger zerolog.Logger, matchers []Matcher) *OutputParser {
	p := &OutputParser{
		matchers: matchers,
		logger:   logger,
	}
	return p
}

// Parse looks at command output, line by line. It uses the defined Matchers to set any appropriate values
func (p *OutputParser) Parse(input *bytes.Buffer) []map[string]string {
	var results []map[string]string

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		row := make(map[string]string)

		// check each possible key match
		for _, m := range p.matchers {
			if m.Match(line) {
				key, err := m.KeyFunc(line)
				if err != nil {
					p.logger.Debug().Err(err).Str("line", line).Msg("key match failed")
					continue
				}

				val, err := m.ValFunc(line)
				if err != nil {
					p.logger.Debug().Err(err).Str("line", line).Msg("value match failed")
					continue
				}

				row[key] = val
				continue
			}
		}

		if len(row) == 0 {
			p.logger.Debug().Str("line", line).Msg("No matched keys")
			continue
		}
		results = append(results, row)

	}
	if err := scanner.Err(); err != nil {
		p.logger.Debug().Err(err).Msg("scanner error")
	}
	return results
}
