package packaging

import "github.com/goreleaser/nfpm/v2/deb"

func BuildDeb(opt Options) error {
	return buildNFPM(opt, deb.Default)
}
