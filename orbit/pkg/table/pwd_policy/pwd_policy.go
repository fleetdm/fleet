//go:build darwin
// +build darwin

package pwd_policy

import (
	"context"
	"errors"
	"fmt"
	"github.com/antchfx/xmlquery"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("maxFailedAttempts"),
		table.IntegerColumn("minLength"),
		table.IntegerColumn("maxPINAgeInDays"),
		table.IntegerColumn("pinHistory"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	uid, gid, err := tbl_common.GetConsoleUidGid()
	if err != nil {
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pwpolicy", "-getaccountpolicies")

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	pwpolicyXMLData := string(out)
	maxFailedAttempts, err := FindMaxFailedAttempts(pwpolicyXMLData)

	return []map[string]string{
		{"maxFailedAttempts": maxFailedAttempts},
		{"minLength": "100"},
		{"maxPINAgeInDays": "100"},
		{"pinHistory": "100"},
	}, nil
}

func FindMaxFailedAttempts(xml string) (maxFailedAttempts string, err error) {
	doc, err := xmlquery.Parse(strings.NewReader(xml))
	if err != nil {
		return "", errors.New("can't parse pwpolicy xml")
	}

	for _, channel := range xmlquery.Find(doc, "//dict") {
		if n := channel.SelectElement("key"); n != nil {
			if n.InnerText() != "policyAttributeMaximumFailedAuthentications" {
				continue
			}
		}
		if n := channel.SelectElement("integer"); n != nil {
			return n.InnerText(), nil
		}
	}
	return "", errors.New("can't find maxFailedAttempts")
}
