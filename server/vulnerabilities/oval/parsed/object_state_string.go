package oval_parsed

import (
	"fmt"
	"regexp"
	"strings"
)

type ObjectStateString string

// NewObjectStateString produces a string with 'op' and 'value' encoded as op|value
func NewObjectStateString(op string, val string) ObjectStateString {
	return ObjectStateString(fmt.Sprintf("%s|%s", op, val))
}

func (sta ObjectStateString) unpack() (OperationType, string) {
	parts := strings.SplitN(string(sta), "|", 2)
	return NewOperationType(parts[0]), parts[1]
}

// Eval evaluates the provided value against the encoded value in sta according to the encoded
// operation.
func (sta ObjectStateString) Eval(other string) (bool, error) {
	op, val := sta.unpack()

	switch op {
	case Equals:
		return val == other, nil
	case NotEqual:
		return val != other, nil
	case CaseInsensitiveEquals:
		return strings.ToLower(val) == strings.ToLower(other), nil
	case CaseInsensitiveNotEqual:
		return strings.ToLower(val) != strings.ToLower(other), nil
	case PatternMatch:
		r, err := regexp.Compile(val)
		if err != nil {
			return false, err
		}
		return r.MatchString(other), nil
	}

	return false, fmt.Errorf("can not compute op %q", op)
}

// ParseKernelVariants extracts kernel variants from the encoded value in sta.
// The encoded value must be of the form of a uname pattern match regex string
// ex.'pattern match|5.15.0-\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k)'
// and returns a slice of the extracted variants.
// ex. ['generic', 'generic-64k', 'generic-lpae', 'lowlatency', 'lowlatency-64k']
func (sta ObjectStateString) ParseKernelVariants() []string {
	op, v := sta.unpack()
	if op != PatternMatch {
		return []string{}
	}

	pattern := `\(-(.*)\)$`

	re := regexp.MustCompile(pattern)

	match := re.FindStringSubmatch(v)
	if len(match) < 2 {
		return []string{}
	}

	// Variants are separated by '|'
	variantsRe := regexp.MustCompile(`\|`)
	variants := variantsRe.Split(match[1], -1)

	// Remove leading '-'
	for i, v := range variants {
		variants[i] = strings.TrimPrefix(v, "-")
	}

	return variants
}
