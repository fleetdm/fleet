package oval_input

type rpmVerifyFileTestStateXML struct {
	Id string `xml:"state_ref,attr"`
}

type rpmVerifyFileTestObjectXML struct {
	Id string `xml:"object_ref,attr"`
}

// RpmVerifyFileTestXML see
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpmverifyfile_test
//
// For RHEL based distros, this test is used to make assertions against the installed OS.
type RpmVerifyFileTestXML struct {
	Id             string                      `xml:"id,attr"`
	CheckExistence string                      `xml:"check_existence,attr"`
	Check          string                      `xml:"check,attr"`
	StateOperator  string                      `xml:"state_operator,attr"`
	Object         rpmVerifyFileTestObjectXML  `xml:"object"`
	States         []rpmVerifyFileTestStateXML `xml:"state"`
}
