package oval_input

// RhelResultXML groups together the different tokens produced from parsing an OVAL file targeting
// RHEL distros.
type RhelResultXML struct {
	Definitions        []DefinitionXML
	RpmVerifyFileTests []RpmVerifyFileTestXML
	RpmInfoTests       []RpmInfoTestXML
	RpmInfoTestStates  []RpmInfoStateXML
	RpmInfoTestObjects []PackageInfoTestObjectXML
	Variables          map[string]ConstantVariableXML
}
