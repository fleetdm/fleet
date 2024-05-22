//go:build darwin
// +build darwin

package tcc_access

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

// TODO - okay to use ~ for userPath?
var userPath, sysPath = "~/Library/Application Support/com.apple.TCC/TCC.db", "/Library/Application Support/com.apple.TCC/TCC.db"

// TODO - instead of getting all rows from tcc.dbs, can we pass condition sent by user query into ?
// to only get desired rows? would elimnate need for filterByColEquality here

// var dbQuery = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access WHERE ?;"

var dbQuery = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access;"

var sqlite3Path = "/usr/bin/sqlite3"

var dbColNames = []string{"service", "client", "client_type", "auth_value", "auth_reason", "last_modified", "policy_id", "indirect_object_identifier", "indirect_object_identifier_type"}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		// TODO: how to add column descriptions?
		table.TextColumn("source"),          // required
		table.TextColumn("service"),         // required
		table.TextColumn("client"),          // required
		table.IntegerColumn("client_type"),  // required
		table.IntegerColumn("auth_value"),   // required
		table.IntegerColumn("auth_reason"),  // required
		table.BigIntColumn("last_modified"), // required
		table.IntegerColumn("policy_id"),
		// TODO - speced as "string" column - meaning text column?
		table.TextColumn("indirect_object_identifier"),
		table.IntegerColumn("indirect_object_identifier_type"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var uRs, sRs, rows []map[string]string
	var err error
	uRs, err = getTCCAccessRows("user", userPath)
	if err != nil {
		return nil, err
	}
	sRs, err = getTCCAccessRows("system", sysPath)
	if err != nil {
		return nil, err
	}

	// TODO - how to get these from WHERE clauses of the query
	var colToVals []colToVal

	rows, err = filterByColEquality(colToVals, append(uRs, sRs...))
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func getTCCAccessRows(source, dbPath string) ([]map[string]string, error) {
	// avoids additional C compilation requirements that would be introduced by using https://github.com/mattn/go-sqlite3
	cmd := exec.Command(sqlite3Path, dbPath, dbQuery)
	dbOut, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	parsedRows, err := parseTCCDbReadOutput(dbOut)
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	rows := buildTableRows(source, parsedRows)

	return rows, nil
}

type colToVal struct {
	col string
	val string
}

func filterByColEquality(colToVals []colToVal, rows []map[string]string) ([]map[string]string, error) {
	// TODO
	return rows, nil
}

func parseTCCDbReadOutput(dbOut []byte) ([][]string, error) {
	// split by newLine for rows, then by "|" for columns
	rawRows := strings.Split(string(dbOut[:]), "\n")

	parsedRows := make([][]string, len(rawRows))

	for _, rawRow := range rawRows {
		parsedRows = append(parsedRows, strings.Split(rawRow, "|"))
	}
	return parsedRows, nil
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
