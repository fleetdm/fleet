package oval_input

// UnixUnameStateXML see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/unix-definitions-schema.html#uname_state
type UnixUnameStateXML struct {
	Id        string         `xml:"id,attr"`
	OSRelease *SimpleTypeXML `xml:"os_release"`
}
