package msrc_input

// XML elements related to the 'vuln' namespace used to describe vulnerabilities, their scores and remediations.

type VulnerabilityXML struct {
	CVE          string                        `xml:"CVE,chardata"`
	Scores       []VulnerabilityScoreXML       `xml:"CVSSScoreSets>ScoreSet"`
	Remediations []VulnerabilityRemediationXML `xml:"Remediations>Remediation"`
}

type VulnerabilityScoreXML struct {
	BaseScore float32 `xml:"BaseScore,chardata"`
	ProductID uint    `xml:"ProductID,chardata"`
}

type VulnerabilityRemediationXML struct {
	Type                 string `xml:"Type,attr"`
	FixedBuild           string `xml:"FixedBuild,chardata"`
	RestartRequired      string `xml:"RestartRequired,chardata"`
	ProductIDs           []uint `xml:"ProductID,chardata"`
	RemediatedBy         uint   `xml:"Description,chardata"`
	RemediationSuperceds uint   `xml:"Supercedence,chardata"`
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
