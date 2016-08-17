package app

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kolide/kolide-ose/sessions"
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

func TestUnauthenticatedPasswordReset(t *testing.T) {
	db := openTestDB(t)
	pool := newMockSMTPConnectionPool()
	r := CreateServer(db, pool, &testLogger{t: t}, &OsqueryLogWriter{Writer: ioutil.Discard}, &OsqueryLogWriter{Writer: ioutil.Discard})
	admin, _ := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)

	{
		response := httptest.NewRecorder()
		body, _ := json.Marshal(&ResetPasswordRequestBody{
			ID: admin.ID,
		})

		buff := new(bytes.Buffer)
		buff.Write(body)
		req, _ := http.NewRequest("POST", "/api/v1/kolide/user/password/reset", buff)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(response, req)
	}

	if len(pool.Emails) != 0 {
		t.Fatal("Email was sent")
	}

	{
		response := httptest.NewRecorder()
		body, _ := json.Marshal(&ResetPasswordRequestBody{
			ID:    admin.ID,
			Email: admin.Email,
		})

		buff := new(bytes.Buffer)
		buff.Write(body)
		req, _ := http.NewRequest("POST", "/api/v1/kolide/user/password/reset", buff)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(response, req)

		if response.Code != 200 {
			t.Fatalf("Response code: %d", response.Code)
		}
	}

	if len(pool.Emails) != 1 {
		t.Fatal("Email was not sent")
	}

	e := pool.Emails[0]
	if e.To[0] != admin.Email {
		t.Fatalf("Email is going to the wrong address: %s", e.To)
	}

	verify := User{
		ID: admin.ID,
	}
	db.Find(&verify).First(&verify)
	if verify.NeedsPasswordReset {
		t.Fatal("User should not need password reset")
	}
}

func TestAuthenticatedPasswordReset(t *testing.T) {
	db := openTestDB(t)
	pool := newMockSMTPConnectionPool()
	r := CreateServer(db, pool, &testLogger{t: t}, &OsqueryLogWriter{Writer: ioutil.Discard}, &OsqueryLogWriter{Writer: ioutil.Discard})
	admin, _ := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)
	request, _ := http.NewRequest("GET", "/", nil)
	writer := newMockResponseWriter()
	sm := sessions.SessionManager{
		Backend: &sessions.GormSessionBackend{DB: db},
		Request: request,
		Writer:  writer,
	}
	sm.MakeSessionForUserID(admin.ID)
	sm.Save()

	adminCookie := writer.Header()["Set-Cookie"][0]

	response := httptest.NewRecorder()
	body, err := json.Marshal(&ResetPasswordRequestBody{
		Username: admin.Username,
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	req, _ := http.NewRequest("POST", "/api/v1/kolide/user/password/reset", buff)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", adminCookie)
	r.ServeHTTP(response, req)

	if response.Code != 200 {
		t.Fatalf("Response code: %d", response.Code)
	}

	if len(pool.Emails) != 1 {
		t.Fatal("Email was not sent")
	}

	e := pool.Emails[0]
	if e.To[0] != admin.Email {
		t.Fatalf("Email is going to the wrong address: %s", e.To)
	}

	verify := User{
		ID: admin.ID,
	}
	db.Find(&verify).First(&verify)
	if !verify.NeedsPasswordReset {
		t.Fatal("User should need password reset")
	}
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
