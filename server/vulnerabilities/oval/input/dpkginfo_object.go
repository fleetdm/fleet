package oval_input

// DpkgInfoObjectXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_object.
type DpkgInfoObjectXML struct {
	Id   string        `xml:"id,attr"`
	Name objectNameXML `xml:"name"`
}
