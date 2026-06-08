package debian

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
	XMLName       xml.Name `xml:"generator"`
	ProductName   string   `xml:"product_name"`
	SchemaVersion string   `xml:"schema_version"`
	Timestamp     string   `xml:"timestamp"`
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
	Debian      Debian      `xml:"metadata>debian"`
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
	Products  []string `xml:"product"`
}

// Reference : >definitions>definition>metadata>reference
type Reference struct {
	XMLName xml.Name `xml:"reference"`
	Source  string   `xml:"source,attr"`
	RefID   string   `xml:"ref_id,attr"`
	RefURL  string   `xml:"ref_url,attr"`
}

// Debian : >definitions>definition>metadata>debian
type Debian struct {
	XMLName  xml.Name `xml:"debian"`
	DSA      string   `xml:"dsa"`
	MoreInfo string   `xml:"moreinfo"`
	Date     string   `xml:"date"`
}

// Tests : >tests
type Tests struct {
	XMLName               xml.Name              `xml:"tests"`
	Textfilecontent54Test Textfilecontent54Test `xml:"textfilecontent54_test"`
	UnameTest             UnameTest             `xml:"uname_test"`
	DpkginfoTest          []DpkginfoTest        `xml:"dpkginfo_test"`
}

// Textfilecontent54Test : >tests>textfilecontent54_test
type Textfilecontent54Test struct {
	Text           string    `xml:",chardata"`
	Check          string    `xml:"check,attr"`
	CheckExistence string    `xml:"check_existence,attr"`
	Comment        string    `xml:"comment,attr"`
	ID             string    `xml:"id,attr"`
	Object         ObjectRef `xml:"object"`
	State          StateRef  `xml:"state"`
}

// UnameTest : >tests>uname_test
type UnameTest struct {
	Text           string    `xml:",chardata"`
	Check          string    `xml:"check,attr"`
	CheckExistence string    `xml:"check_existence,attr"`
	Comment        string    `xml:"comment,attr"`
	ID             string    `xml:"id,attr"`
	Object         ObjectRef `xml:"object"`
}

// DpkginfoTest : >tests>dpkginfo_test
type DpkginfoTest struct {
	Text           string    `xml:",chardata"`
	Check          string    `xml:"check,attr"`
	CheckExistence string    `xml:"check_existence,attr"`
	Comment        string    `xml:"comment,attr"`
	ID             string    `xml:"id,attr"`
	Object         ObjectRef `xml:"object"`
	State          StateRef  `xml:"state"`
}

// ObjectRef :
// >tests>textfilecontent54_test>object-object_ref
// >tests>uname_test>object-object_ref
// >tests>dpkginfo_test>object-object_ref
type ObjectRef struct {
	XMLName   xml.Name `xml:"object"`
	Text      string   `xml:",chardata"`
	ObjectRef string   `xml:"object_ref,attr"`
}

// StateRef :
// >tests>textfilecontent54_test>state-state_ref
// >tests>dpkginfo_test>state-state_ref
type StateRef struct {
	XMLName  xml.Name `xml:"state"`
	Text     string   `xml:",chardata"`
	StateRef string   `xml:"state_ref,attr"`
}

// Objects : >objects
type Objects struct {
	XMLName                 xml.Name                `xml:"objects"`
	Textfilecontent54Object Textfilecontent54Object `xml:"textfilecontent54_object"`
	UnameObject             UnameObject             `xml:"uname_object"`
	DpkginfoObject          []DpkginfoObject        `xml:"dpkginfo_object"`
}

// Textfilecontent54Object : >objects>textfilecontent54_object
type Textfilecontent54Object struct {
	ID       string `xml:"id,attr"`
	Path     string `xml:"path"`
	Filename string `xml:"filename"`
	Pattern  struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
	} `xml:"pattern"`
	Instance struct {
		Text     string `xml:",chardata"`
		Datatype string `xml:"datatype,attr"`
	} `xml:"instance"`
}

// UnameObject : >objects>uname_object
type UnameObject struct {
	ID string `xml:"id,attr"`
}

// DpkginfoObject : >objects>dpkginfo_object
type DpkginfoObject struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name"`
}

// States : >states
type States struct {
	XMLName                xml.Name               `xml:"states"`
	Textfilecontent54State Textfilecontent54State `xml:"textfilecontent54_state"`
	DpkginfoState          []DpkginfoState        `xml:"dpkginfo_state"`
}

// Textfilecontent54State : >states>textfilecontent54_state
type Textfilecontent54State struct {
	ID            string `xml:"id,attr"`
	Subexpression struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
	} `xml:"subexpression"`
}

// DpkginfoState : >states>dpkginfo_state
type DpkginfoState struct {
	ID  string `xml:"id,attr"`
	Evr struct {
		Text      string `xml:",chardata"`
		Datatype  string `xml:"datatype,attr"`
		Operation string `xml:"operation,attr"`
	} `xml:"evr"`
}
