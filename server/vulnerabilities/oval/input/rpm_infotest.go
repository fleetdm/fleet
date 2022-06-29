package oval_input

type rpmInfoTestStateXML struct {
	Id string `xml:"state_ref,attr"`
}

type rpmInfoTestObjectXML struct {
	Id string `xml:"object_ref,attr"`
}

// RpmInfoTestXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpminfo_test
type RpmInfoTestXML struct {
	Id             string                `xml:"id,attr"`
	CheckExistence string                `xml:"check_existence,attr"`
	Check          string                `xml:"check,attr"`
	StateOperator  string                `xml:"state_operator,attr"`
	Object         rpmInfoTestObjectXML  `xml:"object"`
	States         []rpmInfoTestStateXML `xml:"state"`
}
