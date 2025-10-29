// Package variables provides functionality for handling Fleet variables,
// which are template placeholders that can be substituted with actual values
// in various contexts.
package variables

import (
	"regexp"
	"sort"
	"strings"
)

// fleetVariableRegex matches Fleet variables in content.
// It supports two formats:
//   - $FLEET_VAR_NAME - without braces
//   - ${FLEET_VAR_NAME} - with braces
//
// The regex captures the variable name (without the FLEET_VAR_ prefix) in named groups.
var fleetVariableRegex = regexp.MustCompile(`(\$FLEET_VAR_(?P<name1>\w+))|(\${FLEET_VAR_(?P<name2>\w+)})`)

// ProfileDataVariableRegex matches variables present in <data> section of Apple profile, which may cause validation issues.
// This is specific to DigiCert certificate data variables.
var ProfileDataVariableRegex = regexp.MustCompile(`(\$FLEET_VAR_DIGICERT_DATA_(?P<name1>\w+))|(\${FLEET_VAR_DIGICERT_DATA_(?P<name2>\w+)})`)

// Find finds all Fleet variables in the given content and returns them as an array
// without the FLEET_VAR_ prefix. Returns nil if no variables are found.
//
// For example, if the content contains "$FLEET_VAR_HOST_UUID" and "${FLEET_VAR_HOST_EMAIL}",
// this function will return []string{"HOST_EMAIL", "HOST_UUID"}.
func Find(contents string) []string {
	resultSlice := FindKeepDuplicates(contents)
	if len(resultSlice) == 0 {
		return nil
	}
	return Dedupe(resultSlice)
}

// FindKeepDuplicates finds all Fleet variables in the given content and returns them
// as a slice without the FLEET_VAR_ prefix. Duplicates are preserved in the result.
//
// This is useful when you need to know the order or frequency of variable occurrences.
func FindKeepDuplicates(contents string) []string {
	var result []string
	matches := fleetVariableRegex.FindAllStringSubmatch(contents, -1)
	if len(matches) == 0 {
		return nil
	}

	nameToIndex := make(map[string]int, 2)
	for i, name := range fleetVariableRegex.SubexpNames() {
		if name == "" {
			continue
		}
		nameToIndex[name] = i
	}

	for _, match := range matches {
		for _, i := range nameToIndex {
			if match[i] != "" {
				result = append(result, match[i])
			}
		}
	}

	// sort result array by length descending, to ensure longer variables are processed first
	sortedResults := make([]string, len(result))
	copy(sortedResults, result)
	sort.Slice(sortedResults, func(i, j int) bool {
		return len(sortedResults[i]) > len(sortedResults[j])
	})
	return sortedResults
}

// Dedupe removes duplicates from the slice and returns a slice to keep order of variables
func Dedupe(varsWithDupes []string) []string {
	if len(varsWithDupes) == 0 {
		return []string{}
	}

	seenMap := make(map[string]bool, len(varsWithDupes))
	var result []string
	for _, v := range varsWithDupes {
		if !seenMap[v] {
			result = append(result, v)
			seenMap[v] = true
		}
	}
	return result
}

// Replace replaces all occurrences of a specific Fleet variable with the given value.
// The variableName should be provided without the FLEET_VAR_ prefix.
// This function replaces both braced and non-braced versions of the variable.
//
// For example, Replace(content, "HOST_UUID", "123-456") will replace both
// $FLEET_VAR_HOST_UUID and ${FLEET_VAR_HOST_UUID} with "123-456".
func Replace(contents string, variableName string, value string) string {
	// Replace both braced and non-braced versions
	result := strings.ReplaceAll(contents, "$FLEET_VAR_"+variableName, value)
	result = strings.ReplaceAll(result, "${FLEET_VAR_"+variableName+"}", value)
	return result
}

// Contains checks if the given content contains any Fleet variables.
func Contains(contents string) bool {
	return fleetVariableRegex.MatchString(contents)
}

// ContainsBytes checks if the given content contains any Fleet variables (bytes version).
func ContainsBytes(contents []byte) bool {
	return fleetVariableRegex.Match(contents)
}
