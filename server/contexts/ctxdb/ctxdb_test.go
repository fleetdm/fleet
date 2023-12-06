package ctxdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPrimaryRequired(t *testing.T) {
	cases := []struct {
		desc string
		ctx  context.Context
		want bool
	}{
		{"not set", context.Background(), false},
		{"set to true", RequirePrimary(context.Background(), true), true},
		{"set to false", RequirePrimary(context.Background(), false), false},
		{"set to true then false", RequirePrimary(RequirePrimary(context.Background(), true), false), false},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := IsPrimaryRequired(c.ctx)
			require.Equal(t, c.want, got)
		})
	}
}

func TestIsCachedMysqlBypassed(t *testing.T) {
	cases := []struct {
		desc string
		ctx  context.Context
		want bool
	}{
		{"not set", context.Background(), false},
		{"set to true", BypassCachedMysql(context.Background(), true), true},
		{"set to false", BypassCachedMysql(context.Background(), false), false},
		{"set to true then false", BypassCachedMysql(BypassCachedMysql(context.Background(), true), false), false},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := IsCachedMysqlBypassed(c.ctx)
			require.Equal(t, c.want, got)
		})
	}
}
