package main

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
)

// GetSession allows you to get the Session object given a web request. This
// is often used in HTTP handlers as the main entry point into managing and
// manipulating the session
func GetSession(c *gin.Context) *Session {
	return c.MustGet("Session").(*Session)
}

// SessionMiddleware is the middleware used for production session management.
// Tests should use `testSessionMiddleware`, which follows the same pattern,
// but creates a session configured for testing.
func SessionMiddleware(c *gin.Context) {
	CreateSession("Session", sessions.NewCookieStore([]byte("c")))(c)
}

// CreateSessions is a helper which returns a gin.HandlerFunc which creates
// a new session management middleware given the name of the session to manage
// and the session storage mechanism. This is commonly used to generate session
// middleware given a variety of settings in both production and testing
// environments
func CreateSession(name string, store sessions.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := &Session{name, c.Request, store, nil, c.Writer}
		c.Set("Session", s)
		defer context.Clear(c.Request)
		c.Next()
	}
}

// Session is a convenience wrapper around gorilla sessions, which is provided
// by github.com/gorilla/sessions
type Session struct {
	name    string
	request *http.Request
	store   sessions.Store
	session *sessions.Session
	writer  http.ResponseWriter
}

// Session returns the gorilla session from the Session struct and allows you
// to use any of the functionality of the underlying sessions.Session struct
func (s *Session) Session() *sessions.Session {
	if s.session == nil {
		var err error
		s.session, err = s.store.Get(s.request, s.name)
		if err != nil {
			logrus.Error(err.Error())
		}
	}
	return s.session
}

// Set simply sets a session key value pair which will be stored in the
// current session for later usage
func (s *Session) Set(key interface{}, val interface{}) {
	s.Session().Values[key] = val
}

// Get retrieves a session key value pair which has previously been set
func (s *Session) Get(key interface{}) interface{} {
	return s.Session().Values[key]
}

// Delete deletes a session key value pair which has previously been set
func (s *Session) Delete(key interface{}) {
	delete(s.Session().Values, key)
}

// Clear deletes all session key value pairs that are set
func (s *Session) Clear() {
	for key := range s.Session().Values {
		s.Delete(key)
	}
}

// Save writes the session, which is required after altering the session in any
// way
func (s *Session) Save() error {
	return s.Session().Save(s.request, s.writer)
}
