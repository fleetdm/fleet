package apple_mdm

import (
	"testing"
	"text/template"
	"text/template/parse"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/stretchr/testify/assert"
)

// TestProfileTemplatesEscapeFields fails on any bare {{ .Field }} interpolation
// in a profile template, so new string fields can't ship without | xml.
func TestProfileTemplatesEscapeFields(t *testing.T) {
	// integer fields can't carry markup and can't be piped through xml.
	intFields := map[string]struct{}{"AccessRights": {}}

	for name, tmpl := range map[string]*template.Template{
		"OTASCEPTemplate":                                        OTASCEPTemplate,
		"enrollmentProfileMobileconfigTemplate":                  enrollmentProfileMobileconfigTemplate,
		"accountDrivenUserEnrollmentProfileMobileconfigTemplate": accountDrivenUserEnrollmentProfileMobileconfigTemplate,
		"acmeEnrollmentProfileMobileconfigTemplate":              acmeEnrollmentProfileMobileconfigTemplate,
		"mobileconfig.FleetdProfileTemplate":                     mobileconfig.FleetdProfileTemplate,
		"mobileconfig.OTAMobileConfigTemplate":                   mobileconfig.OTAMobileConfigTemplate,
		"mobileconfig.FleetCARootTemplate":                       mobileconfig.FleetCARootTemplate,
	} {
		var walk func(parse.Node)
		walk = func(n parse.Node) {
			switch n := n.(type) {
			case *parse.ListNode:
				if n != nil {
					for _, c := range n.Nodes {
						walk(c)
					}
				}
			case *parse.IfNode:
				walk(n.List)
				walk(n.ElseList)
			case *parse.RangeNode:
				walk(n.List)
				walk(n.ElseList)
			case *parse.WithNode:
				walk(n.List)
				walk(n.ElseList)
			case *parse.ActionNode:
				cmds := n.Pipe.Cmds
				if len(cmds) == 1 && len(cmds[0].Args) == 1 {
					if f, ok := cmds[0].Args[0].(*parse.FieldNode); ok {
						if _, isInt := intFields[f.Ident[len(f.Ident)-1]]; !isInt {
							assert.Failf(t, "unescaped interpolation", "%s: %s is not piped through xml", name, n)
						}
					}
				}
			}
		}
		walk(tmpl.Root)
	}
}
