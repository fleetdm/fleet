package oval_input

// RpmInfoStateXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpminfo_state.
type RpmInfoStateXML struct {
	Id             string         `xml:"id,attr"`
	Name           *simpleTypeXML `xml:"name"`
	Arch           *simpleTypeXML `xml:"arch"`
	Epoch          *simpleTypeXML `xml:"epoch,omitempty"`
	Release        *simpleTypeXML `xml:"release,omitempty"`
	Version        *simpleTypeXML `xml:"version,omitempty"`
	Evr            *simpleTypeXML `xml:"evr"`
	SignatureKeyId *simpleTypeXML `xml:"signature_keyid"`
}
