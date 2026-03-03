package alpine

// SecDB is a struct of alpine secdb
type SecDB struct {
	Distroversion string
	Reponame      string
	Urlprefix     string
	Apkurl        string
	Packages      []struct {
		Pkg struct {
			Name     string
			Secfixes map[string][]string
		}
	}
}
