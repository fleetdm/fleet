package msrc_io

import "testing"

func TestGithubClient(t *testing.T) {
	t.Run("#Bulletins", func(t *testing.T) {
		sut := NewGithubClient(nil, nil, "")
	})
}
