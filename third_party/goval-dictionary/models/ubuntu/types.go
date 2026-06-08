package ubuntu

import "encoding/xml"

// Root : root object
type Root struct {
	XMLName     xml.Name    `xml:"oval_definitions"`
	Generator   Generator   `xml:"generator"`
	Definitions Definitions `xml:"definitions"`
	Tests       Tests       `xml:"tests"`
	Objects     Objects     `xml:"objects"`
	States      States      `xml:"states"`
	Variables   Variables   `xml:"variables"`
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
	Notes       struct {
		Text string `xml:",chardata"`
		Note string `xml:"note"`
	} `xml:"notes"`
	Criteria Criteria `xml:"criteria"`
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
type Advisory struct {
	XMLName    xml.Name `xml:"advisory"`
	Severity   string   `xml:"severity"`
	Rights     string   `xml:"rights"`
	PublicDate string   `xml:"public_date"`
	Refs       []Ref    `xml:"ref"`
	Bugs       []Bug    `xml:"bug"`
}

// Ref : >definitions>definition>metadata>advisory>ref
type Ref struct {
	XMLName xml.Name `xml:"ref"`
	URL     string   `xml:",chardata"`
}

// Bug : >definitions>definition>metadata>advisory>bug
type Bug struct {
	XMLName xml.Name `xml:"bug"`
	URL     string   `xml:",chardata"`
}

// Tests : >tests
type Tests struct {
	XMLName               xml.Name                `xml:"tests"`
	Textfilecontent54Test []Textfilecontent54Test `xml:"textfilecontent54_test"`
}

// Textfilecontent54Test : >tests>textfilecontent54_test
type Textfilecontent54Test struct {
	ID             string    `xml:"id,attr"`
	Check          string    `xml:"check,attr"`
	CheckExistence string    `xml:"check_existence,attr"`
	Comment        string    `xml:"comment,attr"`
	Object         ObjectRef `xml:"object"`
	State          StateRef  `xml:"state"`
}

// ObjectRef : >tests>textfilecontent54_test>object-object_ref
type ObjectRef struct {
	XMLName   xml.Name `xml:"object"`
	Text      string   `xml:",chardata"`
	ObjectRef string   `xml:"object_ref,attr"`
}

// StateRef : >tests>textfilecontent54_test>state-state_ref
type StateRef struct {
	XMLName  xml.Name `xml:"state"`
	Text     string   `xml:",chardata"`
	StateRef string   `xml:"state_ref,attr"`
}

// Objects : >objects
type Objects struct {
	XMLName                 xml.Name                  `xml:"objects"`
	Textfilecontent54Object []Textfilecontent54Object `xml:"textfilecontent54_object"`
}

// Textfilecontent54Object : >objects>textfilecontent54_object
type Textfilecontent54Object struct {
	ID       string `xml:"id,attr"`
	Comment  string `xml:"comment,attr"`
	Path     string `xml:"path"`
	Filename string `xml:"filename"`
	Pattern  struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
		Datatype  string `xml:"datatype,attr"`
		VarRef    string `xml:"var_ref,attr"`
		VarCheck  string `xml:"var_check,attr"`
	} `xml:"pattern"`
	Instance struct {
		Text      string `xml:",chardata"`
		Operation string `xml:"operation,attr"`
		Datatype  string `xml:"datatype,attr"`
	} `xml:"instance"`
}

// States : >states
type States struct {
	XMLName                xml.Name                 `xml:"states"`
	Textfilecontent54State []Textfilecontent54State `xml:"textfilecontent54_state"`
}

// Textfilecontent54State : >states>textfilecontent54_state
type Textfilecontent54State struct {
	ID            string `xml:"id,attr"`
	Comment       string `xml:"comment,attr"`
	Subexpression struct {
		Text      string `xml:",chardata"`
		Datatype  string `xml:"datatype,attr"`
		Operation string `xml:"operation,attr"`
	} `xml:"subexpression"`
}

// Variables : >variables
type Variables struct {
	XMLName          xml.Name           `xml:"variables"`
	ConstantVariable []ConstantVariable `xml:"constant_variable"`
}

// ConstantVariable : >variables>constant_variable
type ConstantVariable struct {
	Text     string   `xml:",chardata"`
	ID       string   `xml:"id,attr"`
	Version  string   `xml:"version,attr"`
	Datatype string   `xml:"datatype,attr"`
	Comment  string   `xml:"comment,attr"`
	Value    []string `xml:"value"`
}

type dpkgInfoTest struct {
	Name         string
	FixedVersion string
}
