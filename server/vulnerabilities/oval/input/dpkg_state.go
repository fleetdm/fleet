package oval_input

// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_state

type ovalSimpleTypeXML struct {
	Datatype string `xml:"datatype"`
	Value    string `xml:",chardata"`
	Op       string `xml:"operation,attr"`
}

type DpkgStateXML struct {
	Id      string             `xml:"id,attr"`
	Name    *ovalSimpleTypeXML `xml:"name"`
	Arch    *ovalSimpleTypeXML `xml:"arch"`
	Epoch   *ovalSimpleTypeXML `xml:"epoch,omitempty"`
	Release *ovalSimpleTypeXML `xml:"release,omitempty"`
	Version *ovalSimpleTypeXML `xml:"version,omitempty"`
	Evr     *ovalSimpleTypeXML `xml:"evr"`
}
