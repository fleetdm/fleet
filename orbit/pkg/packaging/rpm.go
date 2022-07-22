package packaging

import "github.com/goreleaser/nfpm/v2/rpm"

// BuildRPM builds a .rpm package
// Note: this function is not safe for concurrent use
func BuildRPM(opt Options) (string, error) {
	return buildNFPM(opt, rpm.Default)
}
