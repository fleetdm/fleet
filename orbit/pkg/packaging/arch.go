package packaging

import "github.com/goreleaser/nfpm/v2/arch"

// BuildArch builds a .tar.zst package.
// Note: this function is not safe for concurrent use.
func BuildArch(opt Options) (string, error) {
	return buildNFPM(opt, arch.Default)
}
