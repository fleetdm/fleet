package kolide

import (
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
)

func TestGenerateJWT(t *testing.T) {
	tokenString, err := GenerateJWT("4", "")
	token, err := ParseJWT(tokenString, "")
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
