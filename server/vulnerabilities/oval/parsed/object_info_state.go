package oval_parsed

type ObjectInfoState struct {
	Name           *ObjectStateString      `json:",omitempty"`
	Arch           *ObjectStateString      `json:",omitempty"`
	Epoch          *ObjectStateSimpleValue `json:",omitempty"`
	Release        *ObjectStateSimpleValue `json:",omitempty"`
	Version        *ObjectStateSimpleValue `json:",omitempty"`
	Evr            *ObjectStateEvrString   `json:",omitempty"`
	SignatureKeyId *ObjectStateString      `json:",omitempty"`
	ExtendedName   *ObjectStateString      `json:",omitempty"`
	FilePath       *ObjectStateString      `json:",omitempty"`
}
