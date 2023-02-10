//go:build darwin
// +build darwin

// Package firmware_integrity_check implements a table
// to perform an integrity check for Legacy EFI.
package firmware_eficheck_integrity_check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("chip"),
		table.TextColumn("output"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
//
// This table implements the check for macOS 13 5.9 "Ensure Legacy EFI Is Valid and Updating".
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	modelName, err := unix.Sysctl("machdep.cpu.brand_string")
	if err != nil {
		return nil, fmt.Errorf("get CPU brand: %w", err)
	}
	log.Debug().Str("modelName", modelName).Msg("machdep.cpu.brand_string")

	if strings.Contains(modelName, "Apple") {
		// Apple chip, nothing to check.
		return []map[string]string{{
			"chip":   "apple",
			"output": "",
		}}, nil
	}

	// Intel chip
	output, err := exec.Command(
		"/usr/sbin/system_profiler", "SPiBridgeDataType",
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run system_profiler: %w", err)
	}
	log.Debug().Str("output", string(output)).Msg("system_profiler SPiBridgeDataType")

	if strings.Contains(string(output), "Model Name: Apple T2 Security Chip") {
		// Intel T2, nothing to check.
		return []map[string]string{{
			"chip":   "intel-t2",
			"output": "",
		}}, nil
	}

	// Intel T1.
	output, err = exec.Command(
		"/usr/libexec/firmwarecheckers/eficheck/eficheck", "--integrity-check",
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run eficheck: %w", err)
	}
	log.Debug().Str("output", string(output)).Msg("eficheck")

	return []map[string]string{{
		"chip":   "intel-t1",
		"output": string(output),
	}}, nil
}
