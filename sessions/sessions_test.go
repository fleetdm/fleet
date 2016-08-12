package sessions

import (
	"net/http"
	"strings"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
)

func TestGenerateJWT(t *testing.T) {
	jwtKey = "very secure"
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

func TestSessionManager(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := newMockResponseWriter()
	sb := newMockSessionBackend()

	sm := &SessionManager{
		Backend: sb,
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
	tokenString := strings.Split(header, "=")[1]
	token, err := ParseJWT(tokenString)
	if err != nil {
		t.Fatal(err.Error())
	}
	session_key := token.Claims.(jwt.MapClaims)["session_key"].(string)
	session, err := sb.FindKey(session_key)
	if err != nil {
		t.Fatal(err.Error())
	}

	if session.UserID != 1 {
		t.Fatalf("User ID doesn't match. Got: %d", session.UserID)
	}

}
