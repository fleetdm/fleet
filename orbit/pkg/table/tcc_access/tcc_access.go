//go:build darwin
// +build darwin

package tcc_access

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

var (
	testContext   = false
	tccPathPrefix = ""
	tccPathSuffix = "/Library/Application Support/com.apple.TCC/TCC.db"
	dbQuery       = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access;"
	sqlite3Path   = "/usr/bin/sqlite3"
	dbColNames    = []string{"service", "client", "client_type", "auth_value", "auth_reason", "last_modified", "policy_id", "indirect_object_identifier", "indirect_object_identifier_type"}
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		// added here
		table.TextColumn("source"),
		table.IntegerColumn("uid"),
		// derived from a TCC.db
		table.TextColumn("service"),
		table.TextColumn("client"),
		table.IntegerColumn("client_type"),
		table.IntegerColumn("auth_value"),
		table.IntegerColumn("auth_reason"),
		table.BigIntColumn("last_modified"),
		table.IntegerColumn("policy_id"),
		table.TextColumn("indirect_object_identifier"),
		table.IntegerColumn("indirect_object_identifier_type"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.

func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// get all usernames on the mac
	cmd := exec.Command("dscl", ".", "list", "/Users")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	allUsernames := strings.Split(string(out[:]), "\n")
	var usernames []string
	for _, username := range allUsernames {
		if !strings.HasPrefix(username, "_") && username != "nobody" && username != "root" && username != "daemon" && len(username) > 0 {
			usernames = append(usernames, username)
		}
	}
	if testContext {
		usernames = []string{"testUser1", "testUser2"}
	}

	// build rows for every user-level TCC.db
	var rows []map[string]string

	uidConstraintList, ok := queryContext.Constraints["uid"]
	for _, username := range usernames {
		uid, err := getUidFromUsername(username)
		if err != nil {
			return nil, err
		}

		satisfiesUidConstraints := true
		if ok {
			// if there are uid constraints
			satisfiesUidConstraints, err = satisfiesConstraints(uid, uidConstraintList.Constraints)
			if err != nil {
				return nil, err
			}
		}

		if satisfiesUidConstraints {
			tccPath := tccPathPrefix + "/Users/" + username + tccPathSuffix
			uRs, err := getTCCAccessRows(uid, tccPath)
			if err != nil {
				return nil, err
			}
			rows = append(rows, uRs...)

		}
	}

	// and for the system-level TCC.db
	sysSatisfiesUidConstraints := true
	if ok {
		// if there are uid constraints
		sysSatisfiesUidConstraints, err = satisfiesConstraints("0", uidConstraintList.Constraints)
		if err != nil {
			return nil, err
		}
	}
	if sysSatisfiesUidConstraints {
		sRs, err := getTCCAccessRows("0", tccPathPrefix+tccPathSuffix)
		if err != nil {
			return nil, err
		}
		rows = append(rows, sRs...)
	}

	return rows, nil
}

func getTCCAccessRows(uid, tccPath string) ([]map[string]string, error) {
	// querying direclty with sqlite3 avoids additional C compilation requirements that would be introduced by using
	// https://github.com/mattn/go-sqlite3
	cmd := exec.Command(sqlite3Path, tccPath, dbQuery)
	var dbOut bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &dbOut
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Generate failed at `cmd.Run()`%s: %w", stderr.String(), err)
	}

	parsedRows := parseTCCDbReadOutput(dbOut.Bytes())

	rows, err := buildTableRows(uid, parsedRows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func parseTCCDbReadOutput(dbOut []byte) [][]string {
	// split by newLine for rows, then by "|" for columns
	rawRows := strings.Split(string(dbOut[:]), "\n")
	n := len(rawRows)
	// the end of the db response is "\n", making the final row "", which we want to omit
	rawRows = rawRows[:n-1]

	parsedRows := make([][]string, 0, len(rawRows))
	for _, rawRow := range rawRows {
		parsedRows = append(parsedRows, strings.Split(rawRow, "|"))
	}
	return parsedRows
}

func buildTableRows(uid string, parsedRows [][]string) ([]map[string]string, error) {
	source := "system"
	if uid != "0" {
		source = "user"
	}

	var rows []map[string]string
	for _, parsedRow := range parsedRows {
		row := make(map[string]string)
		row["source"] = source
		row["uid"] = uid
		for i, rowColVal := range parsedRow {
			row[dbColNames[i]] = rowColVal
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func getUidFromUsername(username string) (string, error) {
	if testContext {
		testUId, ok := map[string]string{"testUser1": "1", "testUser2": "2"}[username]
		if ok {
			return testUId, nil
		}
		return "", errors.New("Invalid test username")
	}
	cmd := exec.Command("id", username)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("generate failed: %w", err)
	}

	sOut := string(out[:])
	unI := strings.Index(sOut, username)
	return sOut[4 : unI-1], nil
}

func satisfiesConstraints(uid string, constraints []table.Constraint) (bool, error) {
	for _, constraint := range constraints {
		// for each constraint on the column
		switch constraint.Operator {
		case table.OperatorEquals:
			if constraint.Expression != uid {
				return false, nil
			}
		case table.OperatorGreaterThan:
			if constraint.Expression >= uid {
				return false, nil
			}
		case table.OperatorLessThan:
			if constraint.Expression <= uid {
				return false, nil
			}
		case table.OperatorGreaterThanOrEquals:
			if constraint.Expression > uid {
				return false, nil
			}
		case table.OperatorLessThanOrEquals:
			if constraint.Expression < uid {
				return false, nil
			}
		default:
			return false, errors.New("Invalid operator for column 'uid' – only =, <, >, <=, >= are supported")
		}
	}
	return true, nil
}
