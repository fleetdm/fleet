package models

import (
	"testing"
)

func Test_FetchMeta(t *testing.T) {
	var tests = []struct {
		in       FetchMeta
		outdated bool
	}{
		{
			in: FetchMeta{
				SchemaVersion: 1,
			},
			outdated: true,
		},
		{
			in: FetchMeta{
				SchemaVersion: LatestSchemaVersion,
			},
			outdated: false,
		},
	}

	for i, tt := range tests {
		if aout := tt.in.OutDated(); tt.outdated != aout {
			t.Errorf("[%d] outdated expected: %#v\n  actual: %#v\n", i, tt.outdated, aout)
		}
	}
}
