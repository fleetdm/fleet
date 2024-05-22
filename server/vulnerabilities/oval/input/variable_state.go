package oval_input

type VariableStateXML struct {
	Id    string        `xml:"id,attr"`
	Value SimpleTypeXML `xml:"value"`
}
