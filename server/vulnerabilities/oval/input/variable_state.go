package oval_input

// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/independent-definitions-schema.html#variable_state
type VariableStateXML struct {
	Id    string        `xml:"id,attr"`
	Value SimpleTypeXML `xml:"value"`
}
