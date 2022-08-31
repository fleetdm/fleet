package oval_input

// ConstantVariableXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#constant_variable.
type ConstantVariableXML struct {
	Id       string   `xml:"id,attr"`
	DataType string   `xml:"datatype,attr"`
	Values   []string `xml:"value"`
}
