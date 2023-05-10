package file

import "strings"

var InvalidMacOSChars = []rune{':', '\\', '*', '?', '"', '<', '>', '|', 0}

func IsValidMacOSName(fileName string) bool {
	if fileName == "" {
		return false
	}

	for _, char := range InvalidMacOSChars {
		if strings.ContainsRune(fileName, char) {
			return false
		}
	}

	return true
}
