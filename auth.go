package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// ViewerContext is a struct which represents the ability for an execution
// context to participate in certain actions. Most often, a ViewerContext is
// associated with an application user, but a ViewerContext can represent a
// variety of other execution contexts as well (script, test, etc). The main
// purpose of a ViewerContext is to assist in the authorization of sensitive
// actions.
type ViewerContext struct {
	user *User
}

// JWT returns a JWT token in serialized string form given a ViewerContext as
// well as a potential error in the event that things have gone wrong.
func (vc *ViewerContext) JWT() (string, error) {
	return GenerateJWT(vc.user.ID)
}

// IsAdmin indicates whether or not the current user can perform administrative
// actions.
func (vc *ViewerContext) IsAdmin() bool {
	if vc.user != nil {
		return vc.user.Admin && vc.user.Enabled
	}
	return false
}

// UserID is a helper that enables quick access to the user ID of the current
// user.
func (vc *ViewerContext) UserID() (uint, error) {
	if vc.user != nil {
		return vc.user.ID, nil
	}
	return 0, errors.New("No user set")
}

func (vc *ViewerContext) CanPerformActions(db *gorm.DB) bool {
	if vc.user == nil {
		return false
	}

	if !vc.user.Enabled {
		return false
	}

	return true
}

func (vc *ViewerContext) IsUserID(id uint) bool {
	userID, err := vc.UserID()
	if err != nil {
		return false
	}
	if userID == id {
		return true
	}
	return false
}

func (vc *ViewerContext) CanPerformWriteActionOnUser(db *gorm.DB, u *User) bool {
	return vc.CanPerformActions(db) && (vc.IsUserID(u.ID) || vc.IsAdmin())
}

func (vc *ViewerContext) CanPerformReadActionOnUser(db *gorm.DB, u *User) bool {
	return vc.CanPerformActions(db) && (vc.IsUserID(u.ID) || vc.IsAdmin())
}

// GenerateJWT generates a JWT token in serialized string form given a
// ViewerContext as well as a potential error in the event that things have
// gone wrong.
func GenerateJWT(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		// "Not Before": https://tools.ietf.org/html/rfc7519#section-4.1.5
		"nbf": time.Now().UTC().Unix(),
		// "Expiration Time": https://tools.ietf.org/html/rfc7519#section-4.1.4
		"exp": time.Now().UTC().AddDate(0, 2, 0).Unix(),
	})

	return token.SignedString([]byte(config.App.JWTKey))
}

// ParseJWT attempts to parse a JWT token in serialized string form into a
// JWT token in a deserialized jwt.Token struct.
func ParseJWT(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method != jwt.SigningMethodHS256 {
			return nil, errors.New("Unexpected signing method")
		}
		return []byte(config.App.JWTKey), nil
	})
}

// JWTRenewalMiddleware optimistically tries to renew the user's JWT token.
// This allows kolide to have sessions that last forever, assuming that a user
// logs in and uses the application within a reasonable time window (which is
// defined in the JWT token generation method). If anything goes wrong, this
// middleware will back off and defer recovery of the situation to the
// downstream web request.
func JWTRenewalMiddleware(c *gin.Context) {
	session := GetSession(c)
	tokenCookie := session.Get("jwt")
	if tokenCookie == nil {
		c.Next()
		return
	}

	tokenString, ok := tokenCookie.(string)
	if !ok {
		c.Next()
		return
	}

	token, err := ParseJWT(tokenString)
	if err != nil {
		c.Next()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		c.Next()
		return
	}

	userID := uint(claims["user_id"].(float64))

	jwt, err := GenerateJWT(userID)
	if err != nil {
		c.Next()
		return
	}

	session.Set("jwt", jwt)

	c.Next()
}

// GenerateVC generates a ViewerContext given a user struct
func GenerateVC(user *User) *ViewerContext {
	return &ViewerContext{
		user: user,
	}
}

// EmptyVC is a utility which generates an empty ViewerContext. This is often
// used to represent users which are not logged in.
func EmptyVC() *ViewerContext {
	return &ViewerContext{
		user: nil,
	}
}

// VC accepts a web request context and a database handler and attempts
// to parse a user's jwt token out of the active session, validate the token,
// and generate an appropriate ViewerContext given the data in the session.
func VC(c *gin.Context, db *gorm.DB) (*ViewerContext, error) {
	session := GetSession(c)
	tokenCookie := session.Get("jwt")
	if tokenCookie == nil {
		return nil, errors.New("jwt session attribute not set")
	}

	tokenString, ok := tokenCookie.(string)
	if !ok {
		return nil, errors.New("jwt token was not string")
	}

	token, err := ParseJWT(tokenString)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("Invalid token")
	}

	userID := uint(claims["user_id"].(float64))
	var user User
	err = db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}

	return GenerateVC(&user), nil

}

type LoginRequestBody struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var body LoginRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing Login post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	var user User
	err = db.Where("username = ?", body.Username).First(&user).Error
	if err != nil {
		logrus.Debugf("User not found: %s", body.Username)
		UnauthorizedError(c)
		return
	}

	err = user.ValidatePassword(body.Password)
	if err != nil {
		logrus.Debugf("Invalid password for user: %s", body.Username)
		UnauthorizedError(c)
		return
	}

	token, err := GenerateVC(&user).JWT()
	if err != nil {
		logrus.Fatalf("Error generating token: %s", err.Error())
		DatabaseError(c)
		return
	}
	session := GetSession(c)
	session.Set("jwt", token)
	session.Save()

	c.JSON(200, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"name":     user.Name,
		"admin":    user.Admin,
	})
}

func Logout(c *gin.Context) {
	session := GetSession(c)
	session.Clear()
	c.JSON(200, nil)
}

func generateRandomText(length int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func HashPassword(salt, password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword(
		[]byte(fmt.Sprintf("%s%s", salt, password)),
		config.App.BcryptCost,
	)
}

func SaltAndHashPassword(password string) (string, []byte, error) {
	salt := generateRandomText(config.App.SaltLength)
	hashed, err := HashPassword(salt, password)
	return salt, hashed, err
}
