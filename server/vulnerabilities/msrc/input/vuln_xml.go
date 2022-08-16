package msrc_input

// XML elements related to the 'vuln' namespace used to describe vulnerabilities and their remediations.

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

// IncludesVendorFix returns true if the vulnerability has a vendor fix targeting the product
// identified by pID.
func (r *VulnerabilityXML) IncludesVendorFix(pID string) bool {
	for _, re := range r.Remediations {
		if re.Type == "Vendor Fix" {
			for _, vfPID := range re.ProductIDs {
				if vfPID == pID {
					return true
				}
			}
		}
	}

	return false
}
