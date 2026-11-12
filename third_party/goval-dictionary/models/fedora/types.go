package fedora

// Reference has reference information
type Reference struct {
	Href  string `xml:"href,attr" json:"href,omitempty"`
	ID    string `xml:"id,attr" json:"id,omitempty"`
	Title string `xml:"title,attr" json:"title,omitempty"`
	Type  string `xml:"type,attr" json:"type,omitempty"`
}

// Package has affected package information
type Package struct {
	Name     string `xml:"name,attr" json:"name,omitempty"`
	Epoch    string `xml:"epoch,attr" json:"epoch,omitempty"`
	Version  string `xml:"version,attr" json:"version,omitempty"`
	Release  string `xml:"release,attr" json:"release,omitempty"`
	Arch     string `xml:"arch,attr" json:"arch,omitempty"`
	Filename string `xml:"filename" json:"filename,omitempty"`
}

// Updated has updated at
type Updated struct {
	Date string `xml:"date,attr" json:"date,omitempty"`
}

// Issued has issued at
type Issued struct {
	Date string `xml:"date,attr" json:"date,omitempty"`
}

// UpdateInfo has detailed data of Updates
type UpdateInfo struct {
	ID              string      `xml:"id" json:"id,omitempty"`
	Title           string      `xml:"title" json:"title,omitempty"`
	Type            string      `xml:"type,attr" json:"type,omitempty"`
	Issued          Issued      `xml:"issued" json:"issued,omitempty"`
	Updated         Updated     `xml:"updated" json:"updated,omitempty"`
	Severity        string      `xml:"severity" json:"severity,omitempty"`
	Description     string      `xml:"description" json:"description,omitempty"`
	Packages        []Package   `xml:"pkglist>collection>package" json:"packages,omitempty"`
	ModularityLabel string      `json:"modularity_label,omitempty"`
	References      []Reference `xml:"references>reference" json:"references,omitempty"`
	CVEIDs          []string    `json:"cveiDs,omitempty"`
}

// Updates has a list of Update Info
type Updates struct {
	UpdateList []UpdateInfo `xml:"update"`
}
