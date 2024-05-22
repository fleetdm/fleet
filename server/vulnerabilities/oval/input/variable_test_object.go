package oval_input

type variableObjectXML struct {
	Id string `xml:"object_ref"`
}

type variableStateXML struct {
	Id string `xml:"state_ref,attr"`
}

type VariableTestXML struct {
	Id     string             `xml:"id,attr"`
	Object variableObjectXML  `xml:"object"`
	States []variableStateXML `xml:"state"`
}
