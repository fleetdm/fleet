package oval_input

type dpkgInfoTestStateXML struct {
	Id string `xml:"state_ref,attr"`
}

type dpkgInfoTestObjectXML struct {
	Id string `xml:"object_ref,attr"`
}

// DpkgInfoTestXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_test
type DpkgInfoTestXML struct {
	Id             string                 `xml:"id,attr"`
	CheckExistence string                 `xml:"check_existence,attr"`
	Check          string                 `xml:"check,attr"`
	StateOperator  string                 `xml:"state_operator,attr"`
	Object         dpkgInfoTestObjectXML  `xml:"object"`
	States         []dpkgInfoTestStateXML `xml:"state"`
}
