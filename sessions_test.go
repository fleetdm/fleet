package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const testSessionName = "TestSession"

func getTestStore() sessions.Store {
	return sessions.NewCookieStore([]byte("test"))
}

func testSessionMiddleware(c *gin.Context) {
	CreateSession(testSessionName, getTestStore())(c)
}

func TestSessionGetSet(t *testing.T) {
	r := createTestServer()
	r.Use(testSessionMiddleware)
	r.Use(JWTRenewalMiddleware)

	r.GET("/set", func(c *gin.Context) {
		session := GetSession(c)
		session.Set("key", "foobar")
		session.Save()
		c.JSON(200, nil)
	})

	r.GET("/get", func(c *gin.Context) {
		session := GetSession(c)
		if session.Get("key") != "foobar" {
			t.Fatal("Session writing failed")
		}
		c.String(200, "OK")
	})

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	r.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/get", nil)
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)
}

func TestSessionDeleteKey(t *testing.T) {
	r := createTestServer()
	r.Use(testSessionMiddleware)
	r.Use(JWTRenewalMiddleware)

	r.GET("/set", func(c *gin.Context) {
		session := GetSession(c)
		session.Set("key", "foobar")
		session.Save()
		c.JSON(200, nil)
	})

	r.GET("/delete", func(c *gin.Context) {
		session := GetSession(c)
		session.Delete("key")
		session.Save()
		c.JSON(200, nil)
	})

	r.GET("/get", func(c *gin.Context) {
		session := GetSession(c)
		if session.Get("key") != nil {
			t.Fatal("Session deleting failed")
		}
		c.JSON(200, nil)
	})

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	r.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/delete", nil)
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/get", nil)
	req3.Header.Set("Cookie", res2.Header().Get("Set-Cookie"))
	r.ServeHTTP(res3, req3)
}

func TestSessionFlashes(t *testing.T) {
	r := createTestServer()
	r.Use(testSessionMiddleware)
	r.Use(JWTRenewalMiddleware)

	r.GET("/set", func(c *gin.Context) {
		session := GetSession(c)
		session.Session().AddFlash("foobar")
		session.Save()
		c.JSON(200, nil)
	})

	r.GET("/flash", func(c *gin.Context) {
		session := GetSession(c)
		l := len(session.Session().Flashes())
		if l != 1 {
			t.Fatal("Flashes count does not equal 1. Equals ", l)
		}
		session.Save()
		c.JSON(200, nil)
	})

	r.GET("/check", func(c *gin.Context) {
		session := GetSession(c)
		l := len(session.Session().Flashes())
		if l != 0 {
			t.Fatal("flashes count is not 0 after reading. Equals ", l)
		}
		session.Save()
		c.JSON(200, nil)
	})

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	r.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/flash", nil)
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/check", nil)
	req3.Header.Set("Cookie", res2.Header().Get("Set-Cookie"))
	r.ServeHTTP(res3, req3)
}

func TestSessionClear(t *testing.T) {
	data := map[string]string{
		"key": "val",
		"foo": "bar",
	}
	r := createTestServer()
	store := getTestStore()
	r.Use(CreateSession(testSessionName, store))
	r.Use(JWTRenewalMiddleware)

	r.GET("/set", func(c *gin.Context) {
		session := GetSession(c)
		for k, v := range data {
			session.Set(k, v)
		}
		session.Clear()
		session.Save()
		c.JSON(200, nil)
	})

	r.GET("/check", func(c *gin.Context) {
		session := GetSession(c)
		for k, v := range data {
			if session.Get(k) == v {
				t.Fatal("Session clear failed")
			}
		}
		c.JSON(200, nil)
	})

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	r.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/check", nil)
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)
}
