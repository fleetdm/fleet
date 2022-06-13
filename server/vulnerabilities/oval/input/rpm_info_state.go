package oval_input

// RpmInfoStateXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpminfo_state.
type RpmInfoStateXML struct {
	Id             string         `xml:"id,attr"`
	Name           *SimpleTypeXML `xml:"name"`
	Arch           *SimpleTypeXML `xml:"arch"`
	Epoch          *SimpleTypeXML `xml:"epoch,omitempty"`
	Release        *SimpleTypeXML `xml:"release,omitempty"`
	Version        *SimpleTypeXML `xml:"version,omitempty"`
	Evr            *SimpleTypeXML `xml:"evr"`
	SignatureKeyId *SimpleTypeXML `xml:"signature_keyid"`
	ExtendedName   *SimpleTypeXML `xml:"extended_name"`
	Filepath       *SimpleTypeXML `xml:"filepath"`
	Operator       *string        `xml:"operator,attr"`
}
