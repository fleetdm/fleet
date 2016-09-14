package kolide

import (
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestGenerateJWT(t *testing.T) {
	tokenString, err := GenerateJWT("4", "")
	assert.Nil(t, err)

	token, err := ParseJWT(tokenString, "")
	assert.Nil(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.True(t, token.Valid)

	sessionKey := claims["session_key"].(string)
	assert.Equal(t, "4", sessionKey)
}
