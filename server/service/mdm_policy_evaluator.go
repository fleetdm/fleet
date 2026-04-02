package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// EvaluateMDMPolicy evaluates an MDM policy definition against device state data.
// All checks use AND logic — all must pass for the policy to pass.
func EvaluateMDMPolicy(policyID uint, hostID uint, definition fleet.MDMPolicyDefinition, deviceData map[string]fleet.DeviceStateEntry) fleet.MDMPolicyResult {
	now := time.Now()

	// Empty checks = pass (vacuous truth)
	if len(definition.Checks) == 0 {
		return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: true, Timestamp: now}
	}

	for _, check := range definition.Checks {
		// Handle existence checks
		if check.Operator == fleet.MDMPolicyCheckExists {
			if deviceData == nil {
				return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Timestamp: now}
			}
			if _, ok := deviceData[check.Field]; !ok {
				return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Timestamp: now}
			}
			continue
		}
		if check.Operator == fleet.MDMPolicyCheckNotExists {
			if deviceData == nil {
				// Field doesn't exist in nil map — pass
				continue
			}
			if _, ok := deviceData[check.Field]; ok {
				return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Timestamp: now}
			}
			continue
		}

		// All other operators require the field to exist
		if deviceData == nil {
			return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Timestamp: now}
		}
		entry, ok := deviceData[check.Field]
		if !ok {
			return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Timestamp: now}
		}

		passes, err := compareValues(entry.Value, check.Operator, check.Expected)
		if err != nil {
			return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Err: err, Timestamp: now}
		}
		if !passes {
			return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: false, Timestamp: now}
		}
	}

	return fleet.MDMPolicyResult{HostID: hostID, PolicyID: policyID, Passes: true, Timestamp: now}
}

// compareValues compares an actual value against an expected value using the given operator.
func compareValues(actual string, operator fleet.MDMPolicyCheckOperator, expected string) (bool, error) {
	switch operator {
	case fleet.MDMPolicyCheckEq:
		return actual == expected, nil
	case fleet.MDMPolicyCheckNeq:
		return actual != expected, nil
	case fleet.MDMPolicyCheckContains:
		return strings.Contains(actual, expected), nil
	case fleet.MDMPolicyCheckNotContains:
		return !strings.Contains(actual, expected), nil
	case fleet.MDMPolicyCheckVersionGte:
		return compareVersions(actual, expected) >= 0, nil
	case fleet.MDMPolicyCheckVersionLte:
		return compareVersions(actual, expected) <= 0, nil
	case fleet.MDMPolicyCheckGt, fleet.MDMPolicyCheckLt, fleet.MDMPolicyCheckGte, fleet.MDMPolicyCheckLte:
		return compareNumeric(actual, operator, expected)
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareNumeric compares two values numerically. Falls back to lexicographic comparison
// if either value cannot be parsed as a float.
func compareNumeric(actual string, operator fleet.MDMPolicyCheckOperator, expected string) (bool, error) {
	actualF, err1 := strconv.ParseFloat(actual, 64)
	expectedF, err2 := strconv.ParseFloat(expected, 64)

	if err1 != nil || err2 != nil {
		// Fall back to lexicographic comparison
		switch operator {
		case fleet.MDMPolicyCheckGt:
			return actual > expected, nil
		case fleet.MDMPolicyCheckLt:
			return actual < expected, nil
		case fleet.MDMPolicyCheckGte:
			return actual >= expected, nil
		case fleet.MDMPolicyCheckLte:
			return actual <= expected, nil
		}
	}

	switch operator {
	case fleet.MDMPolicyCheckGt:
		return actualF > expectedF, nil
	case fleet.MDMPolicyCheckLt:
		return actualF < expectedF, nil
	case fleet.MDMPolicyCheckGte:
		return actualF >= expectedF, nil
	case fleet.MDMPolicyCheckLte:
		return actualF <= expectedF, nil
	default:
		return false, fmt.Errorf("unexpected numeric operator: %s", operator)
	}
}

// compareVersions compares two version strings (e.g., "17.4.1" vs "17.4").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}
		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
	}
	return 0
}
