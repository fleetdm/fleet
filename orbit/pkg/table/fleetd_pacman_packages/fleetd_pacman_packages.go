package fleetd_pacman_packages

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

const TableName = "fleetd_pacman_packages"

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("name"),
		table.TextColumn("version"),
		table.TextColumn("description"),
		table.TextColumn("arch"),
		table.TextColumn("url"),
		table.TextColumn("licenses"),
		table.TextColumn("groups"),
		table.TextColumn("provides"),
		table.TextColumn("depends_on"),
		table.TextColumn("optional_deps"),
		table.TextColumn("required_by"),
		table.TextColumn("optional_for"),
		table.TextColumn("conflicts_with"),
		table.TextColumn("replaces"),
		table.TextColumn("installed_size"),
		table.TextColumn("packager"),
		table.TextColumn("build_date"),
		table.TextColumn("install_date"),
		table.TextColumn("install_reason"),
		table.TextColumn("install_script"),
		table.TextColumn("validated_by"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var softwareTitles []string

	if software, ok := queryContext.Constraints["name"]; ok {
		for _, c := range software.Constraints {
			if c.Operator == table.OperatorEquals {
				softwareTitles = append(softwareTitles, c.Expression)
			}
		}
	}

	args := []string{"-Qi"}
	args = append(args, softwareTitles...)

	out, err := exec.Command("/usr/bin/pacman", args...).Output()
	if os.IsNotExist(err) {
		// If no package manager, return nothing but don't fail
		return nil, nil
	} else if err != nil {
		// Some other error
		return nil, fmt.Errorf("command failed: %w", err)
	}
	return parsePacmanQiOutput(string(out)), nil
}

func parsePacmanQiOutput(output string) []map[string]string {
	groups := strings.Split(output, "\n\n")
	packages := make([]map[string]string, 0, len(groups))

	for _, group := range groups {
		trimmed := strings.TrimSpace(group)
		if trimmed == "" {
			continue
		}

		pkg := map[string]string{}
		lines := strings.SplitSeq(trimmed, "\n")
		for line := range lines {
			colon := strings.Index(line, ":")
			if colon == -1 || colon == len(line) {
				continue
			}
			key := strings.TrimSpace(line[:colon])
			value := strings.TrimSpace(line[colon+1:])
			if value == "None" {
				value = ""
			}
			switch key {
			case "Name":
				pkg["name"] = value
			case "Version":
				pkg["version"] = value
			case "Description":
				pkg["description"] = value
			case "Architecture":
				pkg["arch"] = value
			case "URL":
				pkg["url"] = value
			case "Licenses":
				pkg["licenses"] = value
			case "Groups":
				pkg["groups"] = value
			case "Provides":
				pkg["provides"] = value
			case "Depends On":
				pkg["depends_on"] = value
			case "Optional Deps":
				pkg["optional_deps"] = value
			case "Required By":
				pkg["required_by"] = value
			case "Optional For":
				pkg["optional_for"] = value
			case "Conflicts With":
				pkg["conflicts_with"] = value
			case "Replaces":
				pkg["replaces"] = value
			case "Installed Size":
				pkg["installed_size"] = value
			case "Packager":
				pkg["packager"] = value
			case "Build Date":
				pkg["build_date"] = value
			case "Install Date":
				pkg["install_date"] = value
			case "Install Reason":
				pkg["install_reason"] = value
			case "Install Script":
				pkg["install_script"] = value
			case "Validated By":
				pkg["validated_by"] = value
			}
		}
		packages = append(packages, pkg)
	}

	return packages
}
