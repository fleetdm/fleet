package oval_input

// UbuntuResultXML groups together the different tokens produced from parsing an OVAL file make for Ubuntu.
type UbuntuResultXML struct {
	Definitions    []DefinitionXML
	PackageTests   []DpkgInfoTestXML
	PackageStates  []DpkgStateXML
	PackageObjects []DpkgObjectXML
	Variables      map[string]ConstantVariableXML
}
