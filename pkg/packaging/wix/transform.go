package wix

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

type node struct {
	XMLName  xml.Name
	Attrs    attrs   `xml:",any,attr"`
	Content  string  `xml:",chardata"`
	Children []*node `xml:",any"`
}

type attrs []*xml.Attr

// Get the value of the attr with the provided name, otherwise returning an
// empty string.
func (a attrs) Get(name string) string {
	for _, attr := range a {
		if attr.Name.Local == name {
			return attr.Value
		}
	}

	return ""
}

func xmlAttr(name, value string) *xml.Attr {
	return &xml.Attr{Name: xml.Name{Local: name}, Value: value}
}

func xmlNode(name string, attrs ...*xml.Attr) *node {
	return &node{
		XMLName: xml.Name{Local: name},
		Attrs:   attrs,
	}
}

func TransformHeat(path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "read file")
	}

	// Eliminate line feeds (they cause extra junk in the result)
	contents = bytes.ReplaceAll(contents, []byte("\r"), []byte(""))

	var n node
	if err := xml.Unmarshal(contents, &n); err != nil {
		return errors.Wrap(err, "unmarshal xml")
	}

	stack := []*node{}
	if err := transform(&n, &stack); err != nil {
		return errors.Wrap(err, "in transform")
	}

	contents, err = xml.MarshalIndent(n, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal xml")
	}

	if err := ioutil.WriteFile(path, contents, 0o600); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func transform(cur *node, stack *[]*node) error {
	// Clear namespace on all elements (generates unnecessarily noisy output if
	// this is not done).
	cur.XMLName.Space = ""

	// Change permissions for all files
	if cur.XMLName.Local == "File" {
		// This SDDL copied directly from osqueryd.exe after a regular
		// osquery MSI install. We assume that osquery is getting the
		// permissions correct and use exactly the same for our files.
		// Using this cryptic string seems to be the only way to disable
		// permission inheritance in a WiX package, so we may not have
		// any option for something more readable.
		//
		// Permissions:
		// Disable inheritance
		// SYSTEM: read/write/execute
		// Administrators: read/write/execute
		// Users: read/execute
		sddl := "O:SYG:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)"
		if strings.HasSuffix(cur.Attrs.Get("Source"), "secret.txt") {
			// This SDDL copied from properly configured file on a Windows 10
			// machine. Permissions are same as above but with access removed
			// for regular users.
			//
			// Permissions:
			// Disable inheritance
			// SYSTEM: read/write/execute
			// Administrators: read/write/execute
			sddl = "O:SYG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA)"

		}
		cur.Children = append(cur.Children, xmlNode(
			"PermissionEx",
			xmlAttr("Sddl", sddl),
		))
	}

	// push current node onto stack
	*stack = append(*stack, cur)
	// Recursively walk the children
	for _, child := range cur.Children {
		if err := transform(child, stack); err != nil {
			return err
		}
	}
	// pop current node from stack
	*stack = (*stack)[:len(*stack)-1]

	return nil
}
