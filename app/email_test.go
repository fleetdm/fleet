package app

import (
	"net/http"
	"strings"
	"testing"
)

func TestGetEmailSubject(t *testing.T) {
	subject, err := GetEmailSubject(PasswordResetEmail)
	if err != nil {
		t.Error(err.Error())
	}
	if subject != "Your Kolide Password Reset Request" {
		t.Errorf("Subject is not as expected: %s", subject)
	}
}

func TestGetEmailBody(t *testing.T) {
	html, text, err := GetEmailBody(PasswordResetEmail, &PasswordResetRequestEmailParameters{
		Name:  "Foo",
		Token: "1234",
	})
	if err != nil {
		t.Error(err.Error())
	}
	for _, body := range [][]byte{html, text} {
		if trimmed := strings.TrimLeft("Hi Foo!", string(body)); trimmed == string(body) {
			t.Errorf("Body didn't start with Hi Foo!: %s", body)
		}
	}
}

type mockResponseWriter struct {
	headers map[string][]string
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers: map[string][]string{},
	}
}

func (w *mockResponseWriter) Header() http.Header {
	return w.headers
}

func (w *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w *mockResponseWriter) WriteHeader(int) {
}

func TestSendEmail(t *testing.T) {
	pool := newMockSMTPConnectionPool()
	err := SendEmail(pool, "mike@kolide.co", "hi", []byte("<p>hey</p>"), []byte("hey"))
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(pool.Emails) != 1 {
		t.Fatalf("Email not sent. %d emails in pool.", len(pool.Emails))
	}

	if string(pool.Emails[0].Text) != "hey" {
		t.Fatalf("Text didn't match. Wanted \"hey\". Got \"%s\"", pool.Emails[0].Text)
	}
}
