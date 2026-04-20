package msi

import (
	"bytes"
	"fmt"
	"io"
	"sort"
)

// WriteMSI creates a complete MSI installer package and writes it to w.
// rootDir is the directory containing all files to be installed.
func WriteMSI(w io.Writer, rootDir string, opts MSIOptions) error {
	pool := NewStringPool()

	// Build all user table data and collect files for the CAB archive.
	tables, cabFiles, err := buildDatabase(pool, rootDir, opts)
	if err != nil {
		return fmt.Errorf("build database: %w", err)
	}

	// Create the embedded CAB archive.
	var cabBuf bytes.Buffer
	if _, err := WriteCab(&cabBuf, cabFiles); err != nil {
		return fmt.Errorf("write cab: %w", err)
	}

	// allTables = user tables only (no _Validation — the MSI engine has built-in
	// knowledge of system tables and doesn't require _Validation to be present).
	allTables := tables

	type streamEntry struct {
		name string
		data []byte
	}
	var streams []streamEntry

	// Encode table data streams first.
	for _, td := range allTables {
		data := EncodeTableData(td, pool)
		if len(data) > 0 {
			streamName := msiEncodeName(td.Schema.Name, true)
			streams = append(streams, streamEntry{name: streamName, data: data})
		}
	}

	// Filter user tables (exclude system tables starting with '_')
	// and sort alphabetically to match msibuild's encoding order.
	var userTablesOnly []*TableData
	for _, td := range allTables {
		if len(td.Schema.Name) == 0 || td.Schema.Name[0] != '_' {
			userTablesOnly = append(userTablesOnly, td)
		}
	}
	sort.Slice(userTablesOnly, func(i, j int) bool {
		return userTablesOnly[i].Schema.Name < userTablesOnly[j].Schema.Name
	})

	// Encode _Tables and _Columns (adds table/column names to pool).
	tablesData := EncodeTablesStream(userTablesOnly, pool)
	columnsData := EncodeColumnsStream(userTablesOnly, pool)
	streams = append(streams, streamEntry{
		name: msiEncodeName("_Tables", true),
		data: tablesData,
	})
	streams = append(streams, streamEntry{
		name: msiEncodeName("_Columns", true),
		data: columnsData,
	})

	// Encode string pool.
	streams = append(streams, streamEntry{
		name: msiEncodeName("_StringPool", true),
		data: pool.EncodePool(),
	})
	streams = append(streams, streamEntry{
		name: msiEncodeName("_StringData", true),
		data: pool.EncodeData(),
	})

	// Encode Summary Information.
	si := NewSummaryInfo(opts.ProductName, opts.Manufacturer, opts.Architecture, opts.ProductVersion)
	streams = append(streams, streamEntry{
		name: "\x05SummaryInformation",
		data: si.Encode(),
	})

	// Add CAB archive as embedded stream. The stream name MUST be MSI-encoded
	// (not plain ASCII) because the Windows MSI engine reads Media.Cabinet value,
	// strips the "#" prefix, then MSI-encodes the result to find the stream in
	// the CFB directory. Using plain "orbit.cab" makes Windows unable to find
	// the cabinet (STG_E_FILENOTFOUND / error 2356).
	streams = append(streams, streamEntry{
		name: msiEncodeName("orbit.cab", false),
		data: cabBuf.Bytes(),
	})

	// Write everything into a CFB container.
	cw := newCFBWriter()
	for _, s := range streams {
		cw.addStream(s.name, s.data)
	}

	if err := cw.writeTo(w); err != nil {
		return fmt.Errorf("write cfb: %w", err)
	}

	return nil
}

