package oval_input

type unixUnameTestStateXML struct {
	Id string `xml:"state_ref,attr"`
}

// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/unix-definitions-schema.html#uname_state
type UnixUnameTestXML struct {
	Id     string                  `xml:"id,attr"`
	States []unixUnameTestStateXML `xml:"state"`
}
