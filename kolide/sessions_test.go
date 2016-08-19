package kolide

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
)

func TestGenerateJWT(t *testing.T) {
	tokenString, err := GenerateJWT("4")
	token, err := ParseJWT(tokenString)
	if err != nil {
		t.Fatal(err.Error())
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		t.Fatal("Token is invalid")
	}

	sessionKey := claims["session_key"].(string)
	if sessionKey != "4" {
		t.Fatalf("Claims are incorrect. session key is %s", sessionKey)
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

func TestSessionManager(t *testing.T) {
	viper.Set("session.cookie_name", "KolideSession")
	r, _ := http.NewRequest("GET", "/", nil)
	w := newMockResponseWriter()
	sb := newMockSessionStore()

	sm := &SessionManager{
		Store:   sb,
		Request: r,
		Writer:  w,
	}

	err := sm.MakeSessionForUserID(1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	err = sm.Save()
	if err != nil {
		t.Fatalf(err.Error())
	}

	header := w.Header().Get("Set-Cookie")
	if header == "" {
		t.Fatal("No cookie was set")
	}
	tokenString := strings.Split(header, "=")[1]
	token, err := ParseJWT(tokenString)
	if err != nil {
		t.Fatal(err.Error())
	}
	session_key := token.Claims.(jwt.MapClaims)["session_key"].(string)
	session, err := sb.FindSessionByKey(session_key)
	if err != nil {
		t.Fatal(err.Error())
	}

	if session.UserID != 1 {
		t.Fatalf("User ID doesn't match. Got: %d", session.UserID)
	}

}

type mockSessionStore struct {
	sessions []*Session
	id       uint
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: []*Session{},
		id:       0,
	}
}

func (s *mockSessionStore) FindSessionByID(id uint) (*Session, error) {
	for _, each := range s.sessions {
		if each.ID == id {
			return each, nil
		}
	}
	return nil, ErrNoActiveSession
}

func (s *mockSessionStore) FindSessionByKey(key string) (*Session, error) {
	for _, each := range s.sessions {
		if each.Key == key {
			return each, nil
		}
	}
	return nil, ErrNoActiveSession
}

func (s *mockSessionStore) FindAllSessionsForUser(id uint) ([]*Session, error) {
	var sessions []*Session
	for _, each := range sessions {
		if each.UserID == id {
			sessions = append(sessions, each)
		}
	}
	return sessions, nil
}

func (s *mockSessionStore) nextID() uint {
	s.id = s.id + 1
	return s.id
}

func (s *mockSessionStore) CreateSessionForUserID(userID uint) (*Session, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:     s.nextID(),
		UserID: userID,
		Key:    base64.StdEncoding.EncodeToString(key),
	}

	err = s.MarkSessionAccessed(session)
	if err != nil {
		return nil, err
	}

	s.sessions = append(s.sessions, session)

	return session, nil
}

func (s *mockSessionStore) DestroySession(session *Session) error {
	var sessions []*Session
	for _, each := range s.sessions {
		if each.ID != session.ID {
			sessions = append(sessions, each)
		}
	}
	s.sessions = sessions
	return nil
}

func (s *mockSessionStore) DestroyAllSessionsForUser(id uint) error {
	var sessions []*Session
	for _, each := range s.sessions {
		if each.UserID != id {
			sessions = append(sessions, each)
		}
	}
	s.sessions = sessions
	return nil
}

func (s *mockSessionStore) MarkSessionAccessed(session *Session) error {
	session.AccessedAt = time.Now().UTC()
	return nil
}
