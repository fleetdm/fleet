//go:build darwin
// +build darwin

package tcc_access

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

var (
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
	// get all usernames and uids on the mac
	usersInfo, err := getUsersInfo()
	if err != nil {
		return nil, err
	}

	var rows []map[string]string
	uidConstraintList, ok := queryContext.Constraints["uid"]

	// build rows for every user-level TCC.db
	for _, userInfo := range usersInfo {
		username, uid := userInfo[0], userInfo[1]
		satisfiesUidConstraints := true

		if ok {
			// there are uid constraints
			satisfiesUidConstraints, err = satisfiesConstraints(uid, uidConstraintList.Constraints)
			if err != nil {
				return nil, err
			}
		}

		if satisfiesUidConstraints {
			tccPath := tccPathPrefix + "/Users/" + username + tccPathSuffix
			if _, err := os.Stat(tccPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					log.Debug().Err(err).Msgf("file for user %s not found: %s", username, tccPath)
					continue
				}
				return nil, err
			}
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
	// querying directly with sqlite3 avoids additional C compilation requirements that would be introduced by using
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
	rawRows := strings.Split(string(dbOut), "\n")
	n := len(rawRows)
	if n == 0 {
		return nil
	}
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
			return false, errors.New("invalid comparison for column 'uid': supported comparisons are `=`, `<`, `>`, `<=`, and `>=`")
		}
	}
	return true, nil
}

func getUsersInfo() ([][]string, error) {
	var parsedFilteredUsersInfo [][]string

	cmd := exec.Command("dscl", ".", "list", "/Users", "UniqueID")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	usersInfo := strings.Split(string(out), "\n")
	for _, userInfo := range usersInfo {
		if len(userInfo) > 0 {
			split := strings.Fields(userInfo)
			uN := split[0]
			// filter for relevant users
			if !strings.HasPrefix(uN, "_") && uN != "nobody" && uN != "root" && uN != "daemon" && len(uN) > 0 {
				parsedFilteredUsersInfo = append(parsedFilteredUsersInfo, split)
			}
		}
	}

	return parsedFilteredUsersInfo, nil
}
