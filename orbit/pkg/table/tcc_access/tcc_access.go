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
	"github.com/rs/zerolog/log"
)

// TODO - QUESTION okay to use ~ for userPath? Seems to work.
// ANSWER Lucas - no, get the actual absolute path
// idea: list user's dir, use those paths.

// QUESTION – case of MULTIPLE USERS include ALL user TCC dbs? How do we know which row belongs to
// which user?
// Lucas - solve by adding column "username", in place of/addition to(?) "source" column?
// If user doesn't specify, include all?
// product answer: add `username` column. QUESTION – what should be this column's value for the
// system-sourced rows?
var userPath, sysPath = "/Users/jacob/Library/Application Support/com.apple.TCC/TCC.db", "/Library/Application Support/com.apple.TCC/TCC.db"

// TODO - QUESTION instead of getting all rows from tcc.dbs, can we pass condition sent by user query into ?
// to only get desired rows? would elimnate need for filterByColEquality here
// see tabl.go > QueryContext, seems like exactly this.
// YES, looks like osquery via Constraints will do this filtering automatically in fact

// var dbQuery = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access WHERE ?;"

var dbQuery = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access;"

var sqlite3Path = "/usr/bin/sqlite3"

var dbColNames = []string{"service", "client", "client_type", "auth_value", "auth_reason", "last_modified", "policy_id", "indirect_object_identifier", "indirect_object_identifier_type"}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("source"),
		// TODO - add a 'username' column that reports the username that a `user`-sourced row comes from
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
	// TODO - check for invalid states:
	// WHERE clause has operation other than equality
	// SELECTed column is invalid
	var uRs, sRs, rows []map[string]string
	var err error
	// TODO - update this to iterate over all existing users, assigning each respective value of
	// "username" to that user's name.
	uRs, err = getTCCAccessRows("user", userPath)
	if err != nil {
		return nil, err
	}
	sRs, err = getTCCAccessRows("system", sysPath)
	if err != nil {
		return nil, err
	}
	rows = append(uRs, sRs...)

	rows, err = filterByColEquality(queryContext.Constraints, rows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func getTCCAccessRows(source, dbPath string) ([]map[string]string, error) {
	// avoids additional C compilation requirements that would be introduced by using
	// https://github.com/mattn/go-sqlite3

	log.Info().Msgf("\n\nsqlite3 path: %v,\ndbPath: %v,\ndbQuery: %v\n", sqlite3Path, dbPath, dbQuery)

	cmd := exec.Command(sqlite3Path, dbPath, dbQuery)
	var dbOut bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &dbOut
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Generate failed at `cmd.Output()`:"+stderr.String()+":%w", err)
		// fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	// dbOut, err := cmd.Output()
	// if err != nil {
	// 	return nil, fmt.Errorf("Generate failed at `cmd.Output()`: %w", err)
	// }

	// Lucas approach from find_cmd_darwin:
	// stdoutPipe, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return nil, fmt.Errorf("create stdout pipe: %w", err)
	// }
	// stderrPipe, err := cmd.StderrPipe()
	// if err != nil {
	// 	return nil, fmt.Errorf("create stderr pipe: %w", err)
	// }

	// if err := cmd.Start(); err != nil {
	// 	return nil, fmt.Errorf("command start failed: %w", err)
	// }

	parsedRows := parseTCCDbReadOutput(dbOut.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	rows := buildTableRows(source, parsedRows)

	return rows, nil
}

func filterByColEquality(constraints map[string]table.ConstraintList, rows []map[string]string) ([]map[string]string, error) {
	// get a simple mapping of columns to the value a row must have for it, if any, as defined by the
	// user-supplied query.
	cVME, err := getColValsMustEqual(constraints)
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	filteredRows := make([]map[string]string, 0, len(rows))
	// for every row
	for _, row := range rows {
		if rowConstraintsSatisfied(row, cVME) {
			filteredRows = append(filteredRows, row)
		}
	}
	return filteredRows, nil
}

func parseTCCDbReadOutput(dbOut []byte) [][]string {
	// split by newLine for rows, then by "|" for columns
	rawRows := strings.Split(string(dbOut[:]), "\n")

	parsedRows := make([][]string, len(rawRows))

	for _, rawRow := range rawRows {
		parsedRows = append(parsedRows, strings.Split(rawRow, "|"))
	}
	return parsedRows
}

func buildTableRows(source string, parsedRows [][]string) []map[string]string {
	var rows []map[string]string
	//  for each row, add "source": source key/val
	for _, parsedRow := range parsedRows {
		row := make(map[string]string)
		row["source"] = source
		for i, rowColVal := range parsedRow {
			row[dbColNames[i]] = rowColVal
		}
		rows = append(rows, row)
	}
	return rows
}

func getColValsMustEqual(constraints map[string]table.ConstraintList) (map[string]string, error) {
	dbColNameSet := make(map[string]struct{}, len(dbColNames))
	for _, name := range dbColNames {
		dbColNameSet[name] = struct{}{}
	}

	cVME := make(map[string]string)
	for col, constraintList := range constraints {
		// TODO - QUESTION, should we check "Affinity" matches column typej–? LUCAs - no, all text

		// check that col is valid column for this table
		// TODO - move this check to top of Generate function
		if _, ok := dbColNameSet[col]; !ok {
			return nil, fmt.Errorf("generate failed: column '%w' not valid for tcc_access ", errors.New(col))
		}
		// TODO - move to top of Generate function
		for _, constraint := range constraintList.Constraints {
			if constraint.Operator != table.OperatorEquals {
				// TODO - QUESTION can we add additional condition options in the future ? Lucas – wait to
				// see how osquery handles these for us
				return nil, errors.New("tcc_access only supports equality operation in WHERE clause")
			}
			cVME[col] = constraint.Expression
		}
	}
	return cVME, nil
}

func rowConstraintsSatisfied(row map[string]string, constraints map[string]string) bool {
	for col, targetVal := range constraints {
		if row[col] != targetVal {
			return false
		}
	}
	return true
}
