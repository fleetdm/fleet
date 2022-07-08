package oval_input

// UbuntuResultXML groups together the different tokens produced from parsing an OVAL file targeting
// Ubuntu distros.
type UbuntuResultXML struct {
	Definitions     []DefinitionXML
	DpkgInfoTests   []DpkgInfoTestXML
	DpkgInfoStates  []DpkgInfoStateXML
	DpkgInfoObjects []PackageInfoTestObjectXML
	Variables       map[string]ConstantVariableXML
}
