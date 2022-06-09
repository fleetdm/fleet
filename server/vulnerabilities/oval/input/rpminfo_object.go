package oval_input

// RpmInfoObjectXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpminfo_object.
type RpmInfoObjectXML struct {
	Id   string        `xml:"id,attr"`
	Name ObjectNameXML `xml:"name"`
}
