//go:build darwin
// +build darwin

package filevault_prk

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("base64_encrypted"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		"bash", "-c",
		`base64 -i /var/db/FileVaultPRK.dat`,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	return []map[string]string{{"base64_encrypted": strings.TrimSpace(string(out))}}, nil
}
