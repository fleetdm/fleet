package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"testing"
	"time"
)

type mockSessionBackend struct {
	sessions []*Session
	id       uint
}

func newMockSessionBackend() *mockSessionBackend {
	return &mockSessionBackend{
		sessions: []*Session{},
		id:       0,
	}
}

func (s *mockSessionBackend) FindID(id uint) (*Session, error) {
	for _, each := range s.sessions {
		if each.ID == id {
			return each, nil
		}
	}
	return nil, ErrNoActiveSession
}

func (s *mockSessionBackend) FindKey(key string) (*Session, error) {
	for _, each := range s.sessions {
		if each.Key == key {
			return each, nil
		}
	}
	return nil, ErrNoActiveSession
}

func (s *mockSessionBackend) FindAllForUser(id uint) ([]*Session, error) {
	var sessions []*Session
	for _, each := range sessions {
		if each.UserID == id {
			sessions = append(sessions, each)
		}
	}
	return sessions, nil
}

func (s *mockSessionBackend) nextID() uint {
	s.id = s.id + 1
	return s.id
}

func (s *mockSessionBackend) Create(userID uint) (*Session, error) {
	key := make([]byte, SessionKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:     s.nextID(),
		UserID: userID,
		Key:    base64.StdEncoding.EncodeToString(key),
	}

	err = s.MarkAccessed(session)
	if err != nil {
		return nil, err
	}

	s.sessions = append(s.sessions, session)

	return session, nil
}

func (s *mockSessionBackend) Destroy(session *Session) error {
	var sessions []*Session
	for _, each := range s.sessions {
		if each.ID != session.ID {
			sessions = append(sessions, each)
		}
	}
	s.sessions = sessions
	return nil
}

func (s *mockSessionBackend) DestroyAllForUser(id uint) error {
	var sessions []*Session
	for _, each := range s.sessions {
		if each.UserID != id {
			sessions = append(sessions, each)
		}
	}
	s.sessions = sessions
	return nil
}

func (s *mockSessionBackend) MarkAccessed(session *Session) error {
	session.AccessedAt = time.Now().UTC()
	return nil
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

func TestFindID(t *testing.T) {

}
