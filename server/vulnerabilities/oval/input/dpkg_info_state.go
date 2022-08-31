package oval_input

// DpkgInfoStateXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_state.
type DpkgInfoStateXML struct {
	Id      string         `xml:"id,attr"`
	Name    *SimpleTypeXML `xml:"name"`
	Arch    *SimpleTypeXML `xml:"arch"`
	Epoch   *SimpleTypeXML `xml:"epoch,omitempty"`
	Release *SimpleTypeXML `xml:"release,omitempty"`
	Version *SimpleTypeXML `xml:"version,omitempty"`
	Evr     *SimpleTypeXML `xml:"evr"`
}
