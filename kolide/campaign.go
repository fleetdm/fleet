package kolide

import (
	"time"

	"github.com/spf13/viper"
)

// CampaignStore manages email campaigns in the database
type CampaignStore interface {
	CreatePassworResetRequest(userID uint, expires time.Time, token string) (*PasswordResetRequest, error)

	DeletePasswordResetRequest(req *PasswordResetRequest) error

	FindPassswordResetByID(id uint) (*PasswordResetRequest, error)

	FindPassswordResetByToken(token string) (*PasswordResetRequest, error)

	FindPassswordResetByTokenAndUserID(token string, id uint) (*PasswordResetRequest, error)
}

// PasswordResetRequest represents a database table for
// Password Reset Requests
type PasswordResetRequest struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	UserID    uint
	Token     string `gorm:"size:1024"`
}

// NewPasswordResetRequest creates a password reset email campaign
func NewPasswordResetRequest(db CampaignStore, userID uint, expires time.Time) (*PasswordResetRequest, error) {

	token, err := generateRandomText(viper.GetInt("smtp.token_key_size"))
	if err != nil {
		return nil, err
	}

	request, err := db.CreatePassworResetRequest(userID, expires, token)
	if err != nil {
		return nil, err
	}

	return request, nil
}
