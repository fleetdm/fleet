package oval_input

type simpleTypeXML struct {
	Datatype string `xml:"datatype"`
	Value    string `xml:",chardata"`
	Op       string `xml:"operation,attr"`
}
