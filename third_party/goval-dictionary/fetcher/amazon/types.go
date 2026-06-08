package amazon

// extrasCatalog is a struct of extras-catalog.json for Amazon Linux 2 Extra Repository
type extrasCatalog struct {
	Topics []struct {
		N            string   `json:"n"`
		Inst         []string `json:"inst,omitempty"`
		Versions     []string `json:"versions"`
		DeprecatedAt string   `json:"deprecated-at,omitempty"`
		Visible      []string `json:"visible,omitempty"`
	} `json:"topics"`
}

// repoMd has repomd data
type repoMd struct {
	RepoList []repo `xml:"data"`
}

// repo has a repo data
type repo struct {
	Type     string   `xml:"type,attr"`
	Location location `xml:"location"`
}

// location has a location of repomd
type location struct {
	Href string `xml:"href,attr"`
}
