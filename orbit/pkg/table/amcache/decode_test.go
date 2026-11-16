//go:build windows

package amcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeTime(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool
	}{
		{"04/05/2023 10:11:12", "1680689472", true},
		{"4/5/2023 10:11:12", "1680689472", true},
		{"11-03-2024", "1730592000", true},
		{"2024-3-11", "1710115200", true},
		{"1/1/1601 00:00:00", "", false},
		{"10.0.22621.1", "", false},
		{"1/1/2999 00:00:00", "", false},
	} {
		got, ok := decodeTime(tc.in)
		assert.Equal(t, tc.want, got, tc.in)
		assert.Equal(t, tc.ok, ok, tc.in)
	}
}

func TestDecodeSha1(t *testing.T) {
	got, ok := decodeSha1("0000F1D2D2F924E990A2FD0B5BB7C5CFD5E1D8E4A3B8")
	assert.True(t, ok)
	assert.Equal(t, "f1d2d2f924e990a2fd0b5bb7c5cfd5e1d8e4a3b8", got)

	got, ok = decodeSha1("f1d2d2f924e990a2fd0b5bb7c5cfd5e1d8e4a3b8")
	assert.True(t, ok)
	assert.Equal(t, "f1d2d2f924e990a2fd0b5bb7c5cfd5e1d8e4a3b8", got)

	for _, in := range []string{"0000zz", "1111f1d2d2f924e990a2fd0b5bb7c5cfd5e1d8e4a3b8", "0000f1d2d2f924e990a2fd0b5bb7c5cfd5e1d8e4a3bz"} {
		_, ok := decodeSha1(in)
		assert.False(t, ok, in)
	}
}

func TestDecodeBool(t *testing.T) {
	for in, want := range map[string]string{"1": "1", "0": "0", "True": "1", "false": "0"} {
		got, ok := decodeBool(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	_, ok := decodeBool("2")
	assert.False(t, ok)
}

func TestDecodeInt(t *testing.T) {
	got, ok := decodeInt("4026531840")
	assert.True(t, ok)
	assert.Equal(t, "4026531840", got)
	for _, in := range []string{"-1", "0x10", "abc"} {
		_, ok := decodeInt(in)
		assert.False(t, ok, in)
	}
}

func TestDecodeMulti(t *testing.T) {
	assert.Equal(t, "a,b,c", decodeMulti(" a , b ,, c "))
	assert.Empty(t, decodeMulti(" , "))
}
