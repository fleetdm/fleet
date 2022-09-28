package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOrGenerateToken(t *testing.T) {
	dir := t.TempDir()
	token, err := loadOrGenerateSecret(dir, "foo")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	token2, err := loadOrGenerateSecret(dir, "foo")
	require.NoError(t, err)
	require.Equal(t, token, token2)
	dir2 := t.TempDir()
	token3, err := loadOrGenerateSecret(dir2, "foo")
	require.NoError(t, err)
	require.NotEmpty(t, token3)
	require.NotEqual(t, token, token3)
	token4, err := loadOrGenerateSecret(dir, "foo2")
	require.NoError(t, err)
	require.NotEmpty(t, token4)
	require.NotEqual(t, token, token3)
}
