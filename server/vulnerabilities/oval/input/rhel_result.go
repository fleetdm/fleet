package oval_input

// RhelResultXML groups together the different tokens produced from parsing an OVAL file targeting
// RHEL distros.
type RhelResultXML struct {
	Definitions        []DefinitionXML
	RpmVerifyFileTests []RpmVerifyFileTest
	RpmInfoTests       []RpmInfoTestXML
	RpmInfoTestStates  []RpmInfoStateXML
	RpmInfoTestObjects []RpmInfoObjectXML
	Variables          map[string]ConstantVariableXML
}
