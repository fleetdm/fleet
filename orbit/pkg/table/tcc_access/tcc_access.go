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
	"github.com/rs/zerolog/log"
)

var (
	tccPathCommon = "/Library/Application Support/com.apple.TCC/TCC.db"
	dbQuery       = "SELECT service, client, client_type, auth_value, auth_reason, last_modified, policy_id, indirect_object_identifier, indirect_object_identifier_type FROM access;"
	sqlite3Path   = "/usr/bin/sqlite3"
	dbColNames    = []string{"service", "client", "client_type", "auth_value", "auth_reason", "last_modified", "policy_id", "indirect_object_identifier", "indirect_object_identifier_type"}
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("source"),
		table.TextColumn("uid"),
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
	// get all human usernames
	// cmd := exec.Command("dscl", ".", "list", "/Users", "|", "grep", "-v", "-e", "'^_'", "-e", "'daemon'", "-e", "'root'", "-e", "'nobody'")
	// log.Info().Msgf("\ncommand to run: %v\n", cmd.String())

	// 2 commands piped together in Go

	// getUserNames := exec.Command("dscl", ".", "list", "/Users")
	// filterHumans := exec.Command("grep", "-v", "-e", "'^_'", "-e", "'daemon'", "-e", "'root'", "-e", "'nobody'")
	// log.Info().Msgf("\nfilter command: %v\n", filterHumans.String())

	// pipe, err := getUserNames.StdoutPipe()
	// if err != nil {
	// 	return nil, fmt.Errorf("generate failed: %w", err)
	// }
	// filterHumans.Stdin = pipe

	// getUserNames.Start()
	// out, err := filterHumans.Output()
	// pipe.Close()

	// if err != nil {
	// 	return nil, fmt.Errorf("generate failed: %w", err)
	// }

	// // out, err := cmd.Output()
	// log.Info().Msgf("\nraw usernames output: %v\n", out)
	// usernames := strings.Split(string(out[:]), "\n")

	// var out bytes.Buffer
	// var stderr bytes.Buffer
	// cmd.Stdout = &out
	// cmd.Stderr = &stderr
	// err := cmd.Run()
	// if err != nil {
	// 	return nil, fmt.Errorf("Generate failed at `cmd.Run()` to get usernames:"+stderr.String()+":%w", err)
	// }
	// log.Info().Msgf("\nraw command out: %v\n", out.Bytes())
	// usernames := strings.Split(string(out.Bytes()[:]), "\n")

	// parse usernames in Go
	cmd := exec.Command("dscl", ".", "list", "/Users")
	out, err := cmd.Output()
	allUsernames := strings.Split(string(out[:]), "\n")
	var usernames []string
	for _, username := range allUsernames {
		if !strings.HasPrefix(username, "_") && username != "nobody" && username != "root" && username != "daemon" {
			usernames = append(usernames, username)
		}
	}

	log.Info().Msgf("\nusernames, split: %v\n", usernames)

	if err != nil {
		return nil, err
	}

	var rows []map[string]string

	for _, username := range usernames {
		log.Info().Msgf("\nusername to get rows: %v\n", username)
		uRs, err := getTCCAccessRows(username)
		log.Info().Msgf("\nuser rows: %v\n", uRs)
		if err != nil {
			return nil, err
		}
		rows = append(rows, uRs...)
	}

	sRs, err := getTCCAccessRows("")
	if err != nil {
		return nil, err
	}
	rows = append(rows, sRs...)

	return rows, nil
}

func getTCCAccessRows(username string) ([]map[string]string, error) {
	// avoids additional C compilation requirements that would be introduced by using
	// https://github.com/mattn/go-sqlite3
	dbPath := tccPathCommon
	if username != "" {
		dbPath = "/Users/" + username + tccPathCommon
	}
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

	rows, err := buildTableRows(username, parsedRows)

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

func buildTableRows(username string, parsedRows [][]string) ([]map[string]string, error) {
	// root's uid, for "system" rows by default
	uid := "0"
	source := "system"
	if username != "" {
		// a user-scoped table, so get its uid
		cmd := exec.Command("id", username)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("generate failed: %w", err)
		}

		sOut := string(out[:])
		unI := strings.Index(sOut, username)
		uid = sOut[4 : unI-1]
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
