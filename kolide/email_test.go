package kolide

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEmailSubject(t *testing.T) {
	subject, err := GetEmailSubject(PasswordResetEmail)
	assert.Nil(t, err)
	assert.Equal(t, "Your Kolide Password Reset Request", subject)
}

func TestGetEmailBody(t *testing.T) {
	html, text, err := GetEmailBody(PasswordResetEmail, PasswordResetRequestEmailParameters{
		Name:  "Foo",
		Token: "1234",
	})
	assert.Nil(t, err)
	for _, body := range [][]byte{html, text} {
		assert.NotEqual(t, string(body), strings.TrimLeft("Hi Foo!", string(body)))
	}
}

func TestSendEmail(t *testing.T) {
	pool := NewMockSMTPConnectionPool()
	err := SendEmail(pool, "mike@kolide.co", "hi", []byte("<p>hey</p>"), []byte("hey"))
	assert.Nil(t, err)

	assert.NotEqual(t, 1, pool.Emails)
	assert.Equal(t, []byte("hey"), pool.Emails[0].Text)
}
