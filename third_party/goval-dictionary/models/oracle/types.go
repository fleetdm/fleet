package oracle

import "encoding/xml"

// Root : root object
type Root struct {
	XMLName     xml.Name    `xml:"oval_definitions"`
	Generator   Generator   `xml:"generator"`
	Definitions Definitions `xml:"definitions"`
	Tests       Tests       `xml:"tests"`
	Objects     Objects     `xml:"objects"`
	States      States      `xml:"states"`
}

// Generator : >generator
type Generator struct {
	XMLName        xml.Name `xml:"generator"`
	ProductName    string   `xml:"product_name"`
	ProductVersion string   `xml:"product_version"`
	SchemaVersion  string   `xml:"schema_version"`
	Timestamp      string   `xml:"timestamp"`
}

// Definitions : >definitions
type Definitions struct {
	XMLName     xml.Name     `xml:"definitions"`
	Definitions []Definition `xml:"definition"`
}

// Definition : >definitions>definition
type Definition struct {
	XMLName     xml.Name    `xml:"definition"`
	ID          string      `xml:"id,attr"`
	Class       string      `xml:"class,attr"`
	Title       string      `xml:"metadata>title"`
	Affecteds   []Affected  `xml:"metadata>affected"`
	References  []Reference `xml:"metadata>reference"`
	Description string      `xml:"metadata>description"`
	Advisory    Advisory    `xml:"metadata>advisory"`
	Criteria    Criteria    `xml:"criteria"`
}

// Criteria : >definitions>definition>criteria
type Criteria struct {
	XMLName    xml.Name    `xml:"criteria"`
	Operator   string      `xml:"operator,attr"`
	Criterias  []Criteria  `xml:"criteria"`
	Criterions []Criterion `xml:"criterion"`
}

// Criterion : >definitions>definition>criteria>*>criterion
type Criterion struct {
	XMLName xml.Name `xml:"criterion"`
	TestRef string   `xml:"test_ref,attr"`
	Comment string   `xml:"comment,attr"`
}

// Affected : >definitions>definition>metadata>affected
type Affected struct {
	XMLName   xml.Name `xml:"affected"`
	Family    string   `xml:"family,attr"`
	Platforms []string `xml:"platform"`
}

// Reference : >definitions>definition>metadata>reference
type Reference struct {
	XMLName xml.Name `xml:"reference"`
	Source  string   `xml:"source,attr"`
	RefID   string   `xml:"ref_id,attr"`
	RefURL  string   `xml:"ref_url,attr"`
}

// Advisory : >definitions>definition>metadata>advisory
// RedHat and Ubuntu OVAL
type Advisory struct {
	XMLName  xml.Name `xml:"advisory"`
	Severity string   `xml:"severity"`
	Rights   string   `xml:"rights"`
	Cves     []Cve    `xml:"cve"`
	Issued   struct {
		Date string `xml:"date,attr"`
	} `xml:"issued"`
}

// Cve : >definitions>definition>metadata>advisory>cve
// RedHat OVAL
type Cve struct {
	XMLName xml.Name `xml:"cve"`
	CveID   string   `xml:",chardata"`
	Href    string   `xml:"href,attr"`
}

// Tests : >tests
type Tests struct {
	XMLName     xml.Name    `xml:"tests"`
	RpminfoTest RpminfoTest `xml:"rpminfo_test"`
}

// RpminfoTest : >tests>rpminfo_test
type RpminfoTest []struct {
	ID      string    `xml:"id,attr"`
	Comment string    `xml:"comment,attr"`
	Check   string    `xml:"check,attr"`
	Object  ObjectRef `xml:"object"`
	State   StateRef  `xml:"state"`
}

// ObjectRef : >tests>rpminfo_test>object-object_ref
type ObjectRef struct {
	XMLName   xml.Name `xml:"object"`
	Text      string   `xml:",chardata"`
	ObjectRef string   `xml:"object_ref,attr"`
}

// StateRef : >tests>rpminfo_test>state-state_ref
type StateRef struct {
	XMLName  xml.Name `xml:"state"`
	Text     string   `xml:",chardata"`
	StateRef string   `xml:"state_ref,attr"`
}

// Objects : >objects
type Objects struct {
	XMLName       xml.Name        `xml:"objects"`
	RpminfoObject []RpminfoObject `xml:"rpminfo_object"`
}

// RpminfoObject : >objects>rpminfo_object
type RpminfoObject struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name"`
}

// States : >states
type States struct {
	XMLName      xml.Name       `xml:"states"`
	RpminfoState []RpminfoState `xml:"rpminfo_state"`
}

// RpminfoState : >states>rpminfo_state
type RpminfoState struct {
	ID             string `xml:"id,attr"`
	SignatureKeyid struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
	} `xml:"signature_keyid"`
	Version struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
	} `xml:"version"`
	Arch struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
	} `xml:"arch"`
	Evr struct {
		Text      string `xml:",chardata"`
		Datatype  string `xml:"datatype,attr"`
		Operation string `xml:"operation,attr"`
	} `xml:"evr"`
}
