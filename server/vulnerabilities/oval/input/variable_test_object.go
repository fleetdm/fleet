package oval_input

type variableObjectXML struct {
	Id string `xml:"object_ref"`
}

type variableStateXML struct {
	Id string `xml:"state_ref,attr"`
}

// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/independent-definitions-schema.html#variable_test
type VariableTestXML struct {
	Id     string             `xml:"id,attr"`
	Object variableObjectXML  `xml:"object"`
	States []variableStateXML `xml:"state"`
}
