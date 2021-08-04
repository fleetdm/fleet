package packaging

import "github.com/goreleaser/nfpm/v2/rpm"

func BuildRPM(opt Options) error {
	return buildNFPM(opt, rpm.Default)
}
