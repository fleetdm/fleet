package datastore

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/kolide/kolide-ose/sessions"
)

var tables = [...]interface{}{
	&kolide.User{},
	&kolide.PasswordResetRequest{},
	&sessions.Session{},
	&kolide.ScheduledQuery{},
	&kolide.Pack{},
	&kolide.DiscoveryQuery{},
	&kolide.Host{},
	&kolide.Label{},
	&kolide.Option{},
	&kolide.Decorator{},
	&kolide.Target{},
	&kolide.DistributedQuery{},
	&kolide.Query{},
	&kolide.DistributedQueryExecution{},
}

type gormDB struct {
	DB *gorm.DB
}

// NewUser creates a new user in the gorm backend
func (orm gormDB) NewUser(user *kolide.User) (*kolide.User, error) {
	err := orm.DB.Create(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// User returns a specific user in the gorm backend
func (orm gormDB) User(username string) (*kolide.User, error) {
	user := &kolide.User{
		Username: username,
	}
	err := orm.DB.Where("username = ?", username).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (orm gormDB) UserByID(id uint) (*kolide.User, error) {
	user := &kolide.User{ID: id}
	err := orm.DB.Where(user).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (orm gormDB) SaveUser(user *kolide.User) error {
	return orm.DB.Save(user).Error
}

func generateRandomText(keySize int) (string, error) {
	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func (orm gormDB) EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*kolide.Host, error) {
	if uuid == "" {
		return nil, errors.New("missing uuid for host enrollment", "programmer error?")
	}
	host := kolide.Host{UUID: uuid}
	err := orm.DB.Where(&host).First(&host).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			// Create new Host
			host = kolide.Host{
				UUID:      uuid,
				HostName:  hostname,
				IPAddress: ip,
				Platform:  platform,
			}

		default:
			return nil, err
		}
	}

	// Generate a new key each enrollment
	host.NodeKey, err = generateRandomText(nodeKeySize)
	if err != nil {
		return nil, err
	}

	// Update these fields if provided
	if hostname != "" {
		host.HostName = hostname
	}
	if ip != "" {
		host.IPAddress = ip
	}
	if platform != "" {
		host.Platform = platform
	}

	if err := orm.DB.Save(&host).Error; err != nil {
		return nil, err
	}

	return &host, nil
}

func (orm gormDB) AuthenticateHost(nodeKey string) (*kolide.Host, error) {
	host := kolide.Host{NodeKey: nodeKey}
	err := orm.DB.Where(&host).First(&host).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			e := errors.NewFromError(
				err,
				http.StatusUnauthorized,
				"Unauthorized",
			)
			// osqueryd expects the literal string "true" here
			e.Extra = map[string]interface{}{"node_invalid": "true"}
			return nil, e
		default:
			return nil, errors.DatabaseError(err)
		}
	}

	return &host, nil
}

func (orm gormDB) UpdateLastSeen(host *kolide.Host) error {
	updateTime := time.Now()
	err := orm.DB.Exec("UPDATE hosts SET updated_at=? WHERE node_key=?", updateTime, host.NodeKey).Error
	if err != nil {
		return errors.DatabaseError(err)
	}
	host.UpdatedAt = updateTime
	return nil
}

func (orm gormDB) CreatePassworResetRequest(userID uint, expires time.Time, token string) (*kolide.PasswordResetRequest, error) {
	campaign := &kolide.PasswordResetRequest{
		UserID:    userID,
		ExpiresAt: expires,
		Token:     token,
	}
	err := orm.DB.Create(campaign).Error
	if err != nil {
		return nil, err
	}

	return campaign, nil
}

func (orm gormDB) DeletePasswordResetRequest(req *kolide.PasswordResetRequest) error {
	err := orm.DB.Delete(req).Error
	return err
}

func (orm gormDB) FindPassswordResetByID(id uint) (*kolide.PasswordResetRequest, error) {
	reset := &kolide.PasswordResetRequest{
		ID: id,
	}
	err := orm.DB.Find(reset).First(reset).Error
	return reset, err
}

func (orm gormDB) FindPassswordResetByToken(token string) (*kolide.PasswordResetRequest, error) {
	reset := &kolide.PasswordResetRequest{
		Token: token,
	}
	err := orm.DB.Find(reset).First(reset).Error
	return reset, err
}

func (orm gormDB) FindPassswordResetByTokenAndUserID(token string, userID uint) (*kolide.PasswordResetRequest, error) {
	reset := &kolide.PasswordResetRequest{
		Token: token,
		ID:    userID,
	}
	err := orm.DB.Find(reset).First(reset).Error
	return reset, err
}

func (orm gormDB) Migrate() error {
	var err error
	for _, table := range tables {
		err = orm.DB.AutoMigrate(table).Error
	}
	return err
}

func (orm gormDB) Drop() error {
	var err error
	for _, table := range tables {
		err = orm.DB.DropTableIfExists(table).Error
	}
	return err
}

// create connection with mysql backend, using a backoff timer and maxAttempts
func openGORM(driver, conn string, maxAttempts int) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		db, err = gorm.Open(driver, conn)
		if err == nil {
			break
		} else {
			if err.Error() == "invalid database source" {
				return nil, err
			}
			// TODO: use a logger
			fmt.Printf("could not connect to mysql: %v\n", err)
			time.Sleep(time.Duration(attempts) * time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql backend, err = %v", err)
	}
	return db, nil
}
