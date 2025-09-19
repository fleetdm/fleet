package packaging

import "github.com/goreleaser/nfpm/v2/arch"

// BuildPkgTarZst builds a .pkg.tar.zst package.
// Note: this function is not safe for concurrent use.
func BuildPkgTarZst(opt Options) (string, error) {
	return buildNFPM(opt, arch.Default)
}
