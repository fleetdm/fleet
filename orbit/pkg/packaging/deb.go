package packaging

import "github.com/goreleaser/nfpm/v2/deb"

// BuildDeb builds a .deb package
// Note: this function is not safe for concurrent use
func BuildDeb(opt Options) (string, error) {
	return buildNFPM(opt, deb.Default)
}
