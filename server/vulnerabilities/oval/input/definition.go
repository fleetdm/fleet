package oval_input

// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#definition

type CriterionXML struct {
	TestId string `xml:"test_ref,attr"`
	Negate string `xml:"negate,attr"`
}

type CriteriaXML struct {
	Operator   string         `xml:"operator,attr"`
	Negate     string         `xml:"negate,attr"`
	Criteriums []CriterionXML `xml:"criterion"`
	Criterias  []CriteriaXML  `xml:"criteria"`
}

type CVEReferenceXML struct {
	Id string `xml:"ref_id,attr"`
}

type DefinitionXML struct {
	Id       string            `xml:"id,attr"`
	CVEs     []CVEReferenceXML `xml:"metadata>reference"`
	Criteria CriteriaXML       `xml:"criteria"`
}
