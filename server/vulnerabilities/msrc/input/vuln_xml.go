package msrc_input

// XML elements related to the 'vuln' namespace used to describe vulnerabilities, their scores and remediations.

type VulnerabilityXML struct {
	CVE          string                        `xml:"CVE"`
	Score        float64                       `xml:"CVSSScoreSets>ScoreSet>BaseScore"`
	Revisions    []RevisionHistoryXML          `xml:"RevisionHistory>Revision"`
	Remediations []VulnerabilityRemediationXML `xml:"Remediations>Remediation"`
}

type RevisionHistoryXML struct {
	Date        string `xml:"Date"`
	Description string `xml:"Description"`
}

type VulnerabilityRemediationXML struct {
	Type            string   `xml:"Type,attr"`
	FixedBuild      string   `xml:"FixedBuild"`
	RestartRequired string   `xml:"RestartRequired"`
	ProductIDs      []string `xml:"ProductID"`
	Description     string   `xml:"Description"`
	Supercedence    string   `xml:"Supercedence"`
}

func (v *VulnerabilityXML) VendorFixes() []VulnerabilityRemediationXML {
	var r []VulnerabilityRemediationXML
	for _, re := range v.Remediations {
		if re.Type == "Vendor Fix" {
			r = append(r, re)
		}
	}
	return r
}
