// Package variables provides functionality for handling Fleet variables,
// which are template placeholders that can be substituted with actual values
// in various contexts.
package variables

import (
	"regexp"
	"strings"
)

// FleetVariableRegex matches Fleet variables in content.
// It supports two formats:
//   - $FLEET_VAR_NAME - without braces
//   - ${FLEET_VAR_NAME} - with braces
//
// The regex captures the variable name (without the FLEET_VAR_ prefix) in named groups.
var FleetVariableRegex = regexp.MustCompile(`(\$FLEET_VAR_(?P<name1>\w+))|(\${FLEET_VAR_(?P<name2>\w+)})`)

// ProfileDataVariableRegex matches variables present in <data> section of Apple profile, which may cause validation issues.
// This is specific to DigiCert certificate data variables.
var ProfileDataVariableRegex = regexp.MustCompile(`(\$FLEET_VAR_DIGICERT_DATA_(?P<name1>\w+))|(\${FLEET_VAR_DIGICERT_DATA_(?P<name2>\w+)})`)

// Find finds all Fleet variables in the given content and returns them as a map
// without the FLEET_VAR_ prefix. Returns nil if no variables are found.
//
// For example, if the content contains "$FLEET_VAR_HOST_UUID" and "${FLEET_VAR_HOST_EMAIL}",
// this function will return map[string]struct{}{"HOST_UUID": {}, "HOST_EMAIL": {}}.
func Find(contents string) map[string]struct{} {
	resultSlice := FindKeepDuplicates(contents)
	if len(resultSlice) == 0 {
		return nil
	}
	return dedupe(resultSlice)
}

// FindKeepDuplicates finds all Fleet variables in the given content and returns them
// as a slice without the FLEET_VAR_ prefix. Duplicates are preserved in the result.
//
// This is useful when you need to know the order or frequency of variable occurrences.
func FindKeepDuplicates(contents string) []string {
	var result []string
	matches := FleetVariableRegex.FindAllStringSubmatch(contents, -1)
	if len(matches) == 0 {
		return nil
	}

	nameToIndex := make(map[string]int, 2)
	for i, name := range FleetVariableRegex.SubexpNames() {
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
	return result
}

// dedupe removes duplicates from the slice and returns a map for O(1) lookups.
func dedupe(varsWithDupes []string) map[string]struct{} {
	result := make(map[string]struct{}, len(varsWithDupes))
	for _, v := range varsWithDupes {
		result[v] = struct{}{}
	}
	return result
}

// Contains checks if the given content contains any Fleet variables.
func Contains(contents string) bool {
	return FleetVariableRegex.MatchString(contents)
}

// ContainsSpecific checks if the given content contains a specific Fleet variable.
// The variableName should be provided without the FLEET_VAR_ prefix.
//
// For example, to check for $FLEET_VAR_HOST_UUID, use ContainsSpecific(content, "HOST_UUID").
func ContainsSpecific(contents string, variableName string) bool {
	// Check both braced and non-braced versions
	nonBraced := "$FLEET_VAR_" + variableName
	braced := "${FLEET_VAR_" + variableName + "}"

	return strings.Contains(contents, nonBraced) || strings.Contains(contents, braced)
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

// ReplaceAll replaces all Fleet variables in the content with their corresponding values
// from the provided map. Variables not in the map are left unchanged.
// The map keys should be variable names without the FLEET_VAR_ prefix.
//
// For example, ReplaceAll(content, map[string]string{"HOST_UUID": "123", "HOST_EMAIL": "test@example.com"})
func ReplaceAll(contents string, values map[string]string) string {
	result := contents
	for varName, value := range values {
		result = Replace(result, varName, value)
	}
	return result
}

// Validate checks if all Fleet variables in the content are valid.
// Returns a slice of invalid variable names (without FLEET_VAR_ prefix).
// An empty slice means all variables are valid.
//
// The validVariables parameter should contain the set of allowed variable names
// without the FLEET_VAR_ prefix.
func Validate(contents string, validVariables map[string]struct{}) []string {
	found := Find(contents)
	if found == nil {
		return nil
	}

	var invalid []string
	for varName := range found {
		if _, ok := validVariables[varName]; !ok {
			invalid = append(invalid, varName)
		}
	}
	return invalid
}

// ExtractVariableName extracts the variable name from a full Fleet variable string.
// It handles both $FLEET_VAR_NAME and ${FLEET_VAR_NAME} formats.
// Returns empty string if the input is not a valid Fleet variable.
//
// For example:
//   - ExtractVariableName("$FLEET_VAR_HOST_UUID") returns "HOST_UUID"
//   - ExtractVariableName("${FLEET_VAR_HOST_EMAIL}") returns "HOST_EMAIL"
//   - ExtractVariableName("not a variable") returns ""
func ExtractVariableName(variable string) string {
	// Trim any whitespace
	variable = strings.TrimSpace(variable)

	// Check if it starts with ${FLEET_VAR_ and ends with }
	if strings.HasPrefix(variable, "${FLEET_VAR_") && strings.HasSuffix(variable, "}") {
		name := strings.TrimPrefix(variable, "${FLEET_VAR_")
		name = strings.TrimSuffix(name, "}")
		return name
	}

	// Check if it starts with $FLEET_VAR_
	if strings.HasPrefix(variable, "$FLEET_VAR_") {
		return strings.TrimPrefix(variable, "$FLEET_VAR_")
	}

	return ""
}

// FormatVariable formats a variable name into the standard Fleet variable format.
// By default, it uses the non-braced format ($FLEET_VAR_NAME).
//
// For example:
//   - FormatVariable("HOST_UUID", false) returns "$FLEET_VAR_HOST_UUID"
//   - FormatVariable("HOST_UUID", true) returns "${FLEET_VAR_HOST_UUID}"
func FormatVariable(variableName string, useBraces bool) string {
	if useBraces {
		return "${FLEET_VAR_" + variableName + "}"
	}
	return "$FLEET_VAR_" + variableName
}
