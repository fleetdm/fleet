package oval_input

// PackageInfoTestObjectXML see
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_object
// and
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpminfo_object
// In the case of 'rpminfo_object' the 'behaviors' child element is not used for testing installed
// rpm packages.
type PackageInfoTestObjectXML struct {
	Id   string        `xml:"id,attr"`
	Name ObjectNameXML `xml:"name"`
}
