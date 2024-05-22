package oval_input

type unixUnameTestObjectXML struct {
	Id string `xml:"object_ref,attr"`
}

type unixUnameTestStateXML struct {
	Id string `xml:"state_ref,attr"`
}

type UnixUnameTestXML struct {
	Id     string                  `xml:"id,attr"`
	States []unixUnameTestStateXML `xml:"state"`
}
