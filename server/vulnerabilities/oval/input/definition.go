package oval_input

// CriterionXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#CriterionType.
type CriterionXML struct {
	TestId string `xml:"test_ref,attr"`
	Negate string `xml:"negate,attr"`
}

// CriteriaXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#CriteriaType.
type CriteriaXML struct {
	Operator   string         `xml:"operator,attr"`
	Negate     string         `xml:"negate,attr"`
	Criteriums []CriterionXML `xml:"criterion"`
	Criterias  []CriteriaXML  `xml:"criteria"`
}

// ReferenceXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#ReferenceType.
type ReferenceXML struct {
	Id string `xml:"ref_id,attr"`
}

// DefinitionXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#definition.
type DefinitionXML struct {
	Id              string         `xml:"id,attr"`
	Vulnerabilities []ReferenceXML `xml:"metadata>reference"`
	Criteria        CriteriaXML    `xml:"criteria"`
}
