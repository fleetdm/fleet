package main

import (
	"sort"
	"encoding/json"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/gosuri/uilive"
	"github.com/kolide/fleet/server/kolide"
)

type outputWriter interface{
	WriteResult(res kolide.DistributedQueryResult) error
}

type resultOutput struct {
	HostIdentifier string              `json:"host"`
	Rows           []map[string]string `json:"rows"`
}

type jsonWriter struct {}

func newJsonWriter() *jsonWriter {
	return &jsonWriter{}
}

func (w *jsonWriter) WriteResult(res kolide.DistributedQueryResult) error {
	out := resultOutput{res.Host.HostName, res.Rows}
	return json.NewEncoder(os.Stdout).Encode(out)
}

type prettyWriter struct {
	results []kolide.DistributedQueryResult
	columns map[string]bool
	writer *uilive.Writer
}

func newPrettyWriter() *prettyWriter{
	return &prettyWriter{
		columns: make(map[string]bool),
		writer: uilive.New(),
	}
}

func (w *prettyWriter) WriteResult(res kolide.DistributedQueryResult) error {
	w.results = append(w.results, res)

	// Recompute columns
	for _, row := range res.Rows {
		delete(row, "host_hostname")
		for col := range row {
			w.columns[col] = true
		}
	}

	columns := []string{}
	for col := range w.columns {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	table := tablewriter.NewWriter(w.writer.Newline())
	table.SetRowLine(true)
	table.SetHeader(append([]string{"hostname"}, columns...))

	// Extract columns from the results in the appropriate order
	for _, res := range w.results {
		for _, row := range res.Rows {
			cols := []string{res.Host.HostName}
			for _, col := range columns {
				cols = append(cols, row[col])
			}
			table.Append(cols)
		}
	}
	table.Render()

	// Actually write the output
	w.writer.Flush()

	return nil
}
