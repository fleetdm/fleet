package oval_input

// DpkgInfoStateXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_state.
type DpkgInfoStateXML struct {
	Id      string         `xml:"id,attr"`
	Name    *simpleTypeXML `xml:"name"`
	Arch    *simpleTypeXML `xml:"arch"`
	Epoch   *simpleTypeXML `xml:"epoch,omitempty"`
	Release *simpleTypeXML `xml:"release,omitempty"`
	Version *simpleTypeXML `xml:"version,omitempty"`
	Evr     *simpleTypeXML `xml:"evr"`
}
