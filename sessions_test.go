package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type MockResponseWriter struct {
}

func (w *MockResponseWriter) Header() http.Header {
	return map[string][]string{}
}

func (w *MockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w *MockResponseWriter) WriteHeader(int) {
}

func TestSessionManagerVC(t *testing.T) {
	db := openTestDB(t)

	admin, err := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	backend := &GormSessionBackend{db}
	session, err := backend.Create(admin.ID)
	if err != nil {
		t.Fatal(err.Error())
	}

	if session.UserID != admin.ID {
		t.Fatal("IDs do not match")
	}

	token, err := GenerateJWT(session.Key)

	cookie := &http.Cookie{
		Name:  CookieName,
		Value: token,
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	req.AddCookie(cookie)

	writer := &MockResponseWriter{}

	sm := &SessionManager{
		request: req,
		writer:  writer,
		backend: backend,
		db:      db,
	}
	vc := sm.VC()

	if !vc.IsAdmin() {
		t.Fatal("User should be admin")
	}

	vcID, _ := vc.UserID()
	if vcID != admin.ID {
		t.Fatal("IDs don't match")
	}
}

func TestSessionCreation(t *testing.T) {
	db := openTestDB(t)
	r := createEmptyTestServer(db)
	admin, _ := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)

	r.GET("/login", func(c *gin.Context) {
		sm := NewSessionManager(c)
		sm.MakeSessionForUser(admin)
		err := sm.Save()
		if err != nil {
			t.Fatal(err.Error())
		}
		c.JSON(200, nil)
	})

	r.GET("/resource", func(c *gin.Context) {
		sm := NewSessionManager(c)
		vc := sm.VC()
		if !vc.IsAdmin() {
			t.Fatal("Request is not admin")
		}
		c.JSON(200, nil)
	})

	r.GET("/nope", func(c *gin.Context) {
		sm := NewSessionManager(c)
		vc := sm.VC()
		if !vc.IsAdmin() {
			t.Fatal("Request is not admin")
		}
		c.JSON(200, nil)
	})

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/login", nil)
	r.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/resource", nil)
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)
}
