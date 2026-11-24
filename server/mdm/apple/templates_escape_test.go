package apple_mdm

import (
	"testing"
	"text/template"
	"text/template/parse"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/stretchr/testify/assert"
)

// TestProfileTemplatesEscapeFields fails on any string field interpolation
// that isn't escaped by a final xml call, so new fields can't ship unescaped.
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
				// any pipeline touching a string field must end in xml, so
				// {{ .X | other }} and {{ printf "%s" .X }} are caught too.
				var hasStringField bool
				for _, cmd := range n.Pipe.Cmds {
					for _, arg := range cmd.Args {
						if f, ok := arg.(*parse.FieldNode); ok {
							if _, isInt := intFields[f.Ident[len(f.Ident)-1]]; !isInt {
								hasStringField = true
							}
						}
					}
				}
				last, ok := n.Pipe.Cmds[len(n.Pipe.Cmds)-1].Args[0].(*parse.IdentifierNode)
				if hasStringField && (!ok || last.Ident != "xml") {
					assert.Failf(t, "unescaped interpolation", "%s: %s does not end in xml", name, n)
				}
			}
		}
		walk(tmpl.Root)
	}
}
