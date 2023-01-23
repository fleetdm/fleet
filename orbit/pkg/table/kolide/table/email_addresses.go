package table

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func EmailAddresses(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("email"),
		table.TextColumn("domain"),
	}
	t := &emailAddressesTable{
		onePasswordAccountsTable: &onePasswordAccountsTable{client: client, logger: logger},
		chromeUserProfilesTable:  &chromeUserProfilesTable{client: client, logger: logger},
	}
	return table.NewPlugin("kolide_email_addresses", columns, t.generateEmailAddresses)
}

type emailAddressesTable struct {
	onePasswordAccountsTable *onePasswordAccountsTable
	chromeUserProfilesTable  *chromeUserProfilesTable
}

func (t *emailAddressesTable) generateEmailAddresses(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	// add results from chrome profiles
	chromeResults, err := t.chromeUserProfilesTable.generate(ctx, queryContext)
	if err != nil {
		return nil, fmt.Errorf("get email addresses from chrome state: %w", err)
	}
	for _, result := range chromeResults {
		email := result["email"]
		// chrome profiles don't require an email skip ones without emails
		if email == "" {
			continue
		}
		results = addEmailToResults(email, results)
	}

	// add results from 1password
	onePassResults, err := t.onePasswordAccountsTable.generate(ctx, queryContext)
	if err != nil {
		return nil, fmt.Errorf("adding email results from 1password config: %w", err)
	}
	for _, onePassResult := range onePassResults {
		results = addEmailToResults(onePassResult["user_email"], results)
	}

	return results, nil
}

func emailDomain(email string) string {
	parts := strings.Split(email, "@")
	switch len(parts) {
	case 0:
		return email
	default:
		return parts[len(parts)-1]
	}
}

func addEmailToResults(email string, results []map[string]string) []map[string]string {
	return append(results, map[string]string{
		"email":  email,
		"domain": emailDomain(email),
	})
}
