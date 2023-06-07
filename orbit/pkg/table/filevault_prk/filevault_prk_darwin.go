//go:build darwin
// +build darwin

package filevault_prk

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

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
	encryptedKey, err := os.ReadFile("/var/db/FileVaultPRK.dat")
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(encryptedKey)

	return []map[string]string{{"base64_encrypted": strings.TrimSpace(encoded)}}, nil
}
