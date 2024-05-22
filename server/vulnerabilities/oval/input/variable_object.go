package oval_input

// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/independent-definitions-schema.html#variable_object
type VariableObjectXML struct {
	Id    string `xml:"id,attr"`
	RefID string `xml:"var_ref"`
}
