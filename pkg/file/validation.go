package file

import "strings"

var InvalidMacOSChars = []rune{':', '\\', '*', '?', '"', '<', '>', '|', 0}

func IsValidMacOSName(fileName string) bool {
	if fileName == "" {
		return false
	}

	return !strings.ContainsAny(fileName, string(InvalidMacOSChars))
}
