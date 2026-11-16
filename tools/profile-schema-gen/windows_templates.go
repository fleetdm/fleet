package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

// windowsTemplatesPath is the profile generator's third-party ADMX template data. It's in this repo,
// so it's read locally and pinned by sha256; no network.
const windowsTemplatesPath = "website/profile-generator/schema/windows-admx-templates.json"

const admxPolicyRoot = "./Device/Vendor/MSFT/Policy/Config/"

type admxTemplateData struct {
	GeneratedAt string `json:"generatedAt"`
	Nodes       []struct {
		Template   string   `json:"admxTemplate"`
		LocURI     string   `json:"locUri"`
		Scopes     []string `json:"scopes"`
		Format     string   `json:"format"`
		Deprecated bool     `json:"deprecated"`
	} `json:"nodes"`
}

// addWindowsTemplates adds the template policies to the schema as source admx_template, once per
// scope they apply to, and returns the pin for the data file. The paths use the AppNames the
// profile generator installs the templates under.
func addWindowsTemplates(ctx context.Context, logger *slog.Logger, path string, schema *profilevalidation.ProfileValidationSchema) (profilevalidation.WindowsTemplatesVersion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("read ADMX template data: %w", err)
	}
	var tmpl admxTemplateData
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(tmpl.Nodes) == 0 {
		return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("%s has no policies", path)
	}

	perTemplate := map[string]int{}
	for _, n := range tmpl.Nodes {
		rel, ok := strings.CutPrefix(n.LocURI, admxPolicyRoot)
		if !ok || rel == "" {
			return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("%s: policy %q isn't under %s", path, n.LocURI, admxPolicyRoot)
		}
		if len(n.Scopes) == 0 {
			return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("%s: policy %s has no scopes", path, n.LocURI)
		}
		for _, scope := range n.Scopes {
			node := &profilevalidation.WindowsNode{
				Format: profilevalidation.WindowsFormats{profilevalidation.WindowsFormat(n.Format)},
				Source: profilevalidation.WindowsSourceADMXTemplate,
			}
			var locURI string
			switch scope {
			case "Device":
				node.Scope, locURI = profilevalidation.WindowsScopeDevice, n.LocURI
			case "User":
				node.Scope, locURI = profilevalidation.WindowsScopeUser, "./User/Vendor/MSFT/Policy/Config/"+rel
			default:
				return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("%s: policy %s has unknown scope %q", path, n.LocURI, scope)
			}
			if n.Deprecated {
				node.Deprecated = &profilevalidation.WindowsDeprecated{}
			}
			// Template areas contain "~" and DDF areas never do, so a clash means bad data.
			if _, ok := schema.Windows[locURI]; ok {
				return profilevalidation.WindowsTemplatesVersion{}, fmt.Errorf("%s: policy %s is already in the schema", path, locURI)
			}
			schema.Windows[locURI] = node
			perTemplate[n.Template]++
		}
	}

	args := []any{"path", path, "policies", len(tmpl.Nodes), "generated_at", tmpl.GeneratedAt}
	for _, name := range slices.Sorted(maps.Keys(perTemplate)) {
		args = append(args, name, perTemplate[name])
	}
	logger.InfoContext(ctx, "added ADMX template policies", args...)
	return profilevalidation.WindowsTemplatesVersion{SHA256: sha256Hex(data), GeneratedAt: tmpl.GeneratedAt}, nil
}
