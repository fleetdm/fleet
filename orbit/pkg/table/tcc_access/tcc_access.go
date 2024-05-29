//go:build darwin
// +build darwin

package tcc_access

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

var userPath, sysPath = "/Users/jacob/Library/Application Support/com.apple.TCC/TCC.db", "/Library/Application Support/com.apple.TCC/TCC.db"

var dbQuery = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access;"

var sqlite3Path = "/usr/bin/sqlite3"

var dbColNames = []string{"service", "client", "client_type", "auth_value", "auth_reason", "last_modified", "policy_id", "indirect_object_identifier", "indirect_object_identifier_type"}

// TODO - add "username"
var constructedColNames = []string{"source"}

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
	var err error
	// TODO - update this to iterate over all existing users, assigning each respective value of
	// "username" to that user's name.
	uRs, err := getTCCAccessRows("user", userPath)
	if err != nil {
		return nil, err
	}
	sRs, err := getTCCAccessRows("system", sysPath)
	if err != nil {
		return nil, err
	}
	return append(uRs, sRs...), nil
}

func getTCCAccessRows(source, dbPath string) ([]map[string]string, error) {
	// avoids additional C compilation requirements that would be introduced by using
	// https://github.com/mattn/go-sqlite3
	cmd := exec.Command(sqlite3Path, dbPath, dbQuery)
	var dbOut bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &dbOut
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Generate failed at `cmd.Run()`:"+stderr.String()+":%w", err)
	}

	parsedRows := parseTCCDbReadOutput(dbOut.Bytes())

	rows := buildTableRows(source, parsedRows)

	return rows, nil
}

func parseTCCDbReadOutput(dbOut []byte) [][]string {
	// split by newLine for rows, then by "|" for columns
	rawRows := strings.Split(string(dbOut[:]), "\n")
	n := len(rawRows)
	for len(rawRows[n-1]) == 0 {
		// if the end of the db response is "\n", the final row will be "", which we want to omit
		rawRows = rawRows[:n-1]
		n = len(rawRows)
	}

	var parsedRows [][]string
	for _, rawRow := range rawRows {
		parsedRows = append(parsedRows, strings.Split(rawRow, "|"))
	}
	return parsedRows
}

func buildTableRows(source string, parsedRows [][]string) []map[string]string {
	var rows []map[string]string
	//  for each row, add "source": source key/val
	// TODO - add "username"
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
