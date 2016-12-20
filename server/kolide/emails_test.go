package kolide

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateProcessor(t *testing.T) {
	mailer := PasswordResetMailer{
		KolideServerURL: "https://localhost.com:8080",
		Token:           "12345",
	}

	out, err := mailer.Message()
	require.Nil(t, err)
	assert.NotNil(t, out)
}
