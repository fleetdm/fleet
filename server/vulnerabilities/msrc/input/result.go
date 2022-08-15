package msrc_input

type ResultXML struct {
	Vulnerabities []VulnerabilityXML
	Products      map[string]ProductXML
}
