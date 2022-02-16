//go:build ignore
// +build ignore

package gorules

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

func fmtErrorfWithoutArgs(m dsl.Matcher) {
	m.Match(`fmt.Errorf($msg)`).
		Report(`fmt.Errorf: change for errors.New($msg)`).
		Suggest(`errors.New($msg)`)
}

func createHttpClient(m dsl.Matcher) {
	m.Match(
		`http.Client{$*_}`,
		`new(http.Client)`,
		`http.Transport{$*_}`,
		`new(http.Transport)`,
	).Report(`http.Client: use fleethttp.NewClient instead`)
}
