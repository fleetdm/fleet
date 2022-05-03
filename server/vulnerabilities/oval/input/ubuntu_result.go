package oval_input

type UbuntuResultXML struct {
	Definitions    []DefinitionXML
	PackageTests   []DpkgInfoTestXML
	PackageStates  []DpkgStateXML
	PackageObjects []DpkgObjectXML
	Variables      map[string]ConstantVariableXML
}
