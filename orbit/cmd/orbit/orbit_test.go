package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOrGenerateToken(t *testing.T) {
	dir := t.TempDir()
	token, err := loadOrGenerateToken(dir)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	token2, err := loadOrGenerateToken(dir)
	require.NoError(t, err)
	require.Equal(t, token, token2)
	dir2 := t.TempDir()
	token3, err := loadOrGenerateToken(dir2)
	require.NoError(t, err)
	require.NotEmpty(t, token3)
	require.NotEqual(t, token, token3)
}
