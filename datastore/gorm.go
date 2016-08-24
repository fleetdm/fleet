package datastore

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // db driver
	_ "github.com/mattn/go-sqlite3"    // db driver

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/spf13/viper"
)

var tables = [...]interface{}{
	&kolide.User{},
	&kolide.PasswordResetRequest{},
	&kolide.Session{},
	&kolide.Pack{},
	&kolide.PackQuery{},
	&kolide.PackTarget{},
	&kolide.Host{},
	&kolide.Label{},
	&kolide.LabelQueryExecution{},
	&kolide.Option{},
	&kolide.Target{},
	&kolide.DistributedQueryCampaign{},
	&kolide.DistributedQueryCampaignTarget{},
	&kolide.Query{},
	&kolide.DistributedQueryExecution{},
}

type gormDB struct {
	DB     *gorm.DB
	Driver string
}

func (orm gormDB) Name() string {
	return "gorm"
}

func (orm gormDB) Migrate() error {
	for _, table := range tables {
		if err := orm.DB.AutoMigrate(table).Error; err != nil {
			return err
		}
	}

	// Have to manually add indexes. Yuck!
	orm.DB.Model(&kolide.LabelQueryExecution{}).AddUniqueIndex("idx_lqe_label_host", "label_id", "host_id")

	return nil
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

// UserByID returns a datastore user given a user ID
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

func (orm gormDB) MarkHostSeen(host *kolide.Host, t time.Time) error {
	updateTime := time.Now()
	err := orm.DB.Exec("UPDATE hosts SET updated_at=? WHERE node_key=?", t, host.NodeKey).Error
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

func (orm gormDB) FindPassswordResetsByUserID(id uint) ([]*kolide.PasswordResetRequest, error) {
	reset := &kolide.PasswordResetRequest{
		UserID: id,
	}
	var resets []*kolide.PasswordResetRequest
	err := orm.DB.Find(reset).First(&resets).Error
	return resets, err
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
		Token:  token,
		UserID: userID,
	}
	err := orm.DB.Find(reset).First(reset).Error
	return reset, err
}

func (orm gormDB) validateSession(session *kolide.Session) error {
	sessionLifeSpan := viper.GetFloat64("session.expiration_seconds")
	if sessionLifeSpan == 0 {
		return nil
	}
	if time.Since(session.AccessedAt).Seconds() >= sessionLifeSpan {
		err := orm.DB.Delete(session).Error
		if err != nil {
			return err
		}
		return kolide.ErrSessionExpired
	}

	err := orm.MarkSessionAccessed(session)
	if err != nil {
		return err
	}

	return nil
}

func (orm gormDB) FindSessionByID(id uint) (*kolide.Session, error) {
	session := &kolide.Session{
		ID: id,
	}

	err := orm.DB.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, kolide.ErrNoActiveSession
		default:
			return nil, err
		}
	}

	err = orm.validateSession(session)
	if err != nil {
		return nil, err
	}

	return session, nil

}

func (orm gormDB) FindSessionByKey(key string) (*kolide.Session, error) {
	session := &kolide.Session{
		Key: key,
	}

	err := orm.DB.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, kolide.ErrNoActiveSession
		default:
			return nil, err
		}
	}

	err = orm.validateSession(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (orm gormDB) FindAllSessionsForUser(id uint) ([]*kolide.Session, error) {
	var sessions []*kolide.Session
	err := orm.DB.Where("user_id = ?", id).Find(&sessions).Error
	return sessions, err
}

func (orm gormDB) CreateSessionForUserID(userID uint) (*kolide.Session, error) {
	sessionKeySize := viper.GetInt("session.key_size")
	if sessionKeySize == 0 {
		sessionKeySize = 24
	}
	key := make([]byte, sessionKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	session := kolide.Session{
		UserID: userID,
		Key:    base64.StdEncoding.EncodeToString(key),
	}

	err = orm.DB.Create(&session).Error
	if err != nil {
		return nil, err
	}

	err = orm.MarkSessionAccessed(&session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (orm gormDB) DestroySession(session *kolide.Session) error {
	return orm.DB.Delete(session).Error
}

func (orm gormDB) DestroyAllSessionsForUser(id uint) error {
	return orm.DB.Delete(&kolide.Session{}, "user_id = ?", id).Error
}

func (orm gormDB) MarkSessionAccessed(session *kolide.Session) error {
	session.AccessedAt = time.Now().UTC()
	return orm.DB.Save(session).Error
}

func (orm gormDB) NewQuery(query *kolide.Query) error {
	if query == nil {
		return errors.New(
			"error creating query",
			"nil pointer passed to NewQuery",
		)
	}
	return orm.DB.Create(query).Error
}

func (orm gormDB) SaveQuery(query *kolide.Query) error {
	if query == nil {
		return errors.New(
			"error saving query",
			"nil pointer passed to SaveQuery",
		)
	}
	return orm.DB.Save(query).Error
}

func (orm gormDB) DeleteQuery(query *kolide.Query) error {
	if query == nil {
		return errors.New(
			"error deleting query",
			"nil pointer passed to DeleteQuery",
		)
	}
	return orm.DB.Delete(query).Error
}

func (orm gormDB) Query(id uint) (*kolide.Query, error) {
	query := &kolide.Query{
		ID: id,
	}
	err := orm.DB.Where(query).First(query).Error
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (orm gormDB) Queries() ([]*kolide.Query, error) {
	var queries []*kolide.Query
	err := orm.DB.Find(&queries).Error
	return queries, err
}

func (orm gormDB) NewLabel(label *kolide.Label) error {
	if label == nil {
		return errors.New(
			"error creating label",
			"nil pointer passed to NewLabel",
		)
	}
	return orm.DB.Create(label).Error
}

func (orm gormDB) LabelQueriesForHost(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
	if host == nil {
		return nil, errors.New(
			"error finding host queries",
			"nil pointer passed to LabelQueriesForHost",
		)
	}
	rows, err := orm.DB.Raw(`
SELECT q.id, q.query
FROM labels l JOIN queries q
ON l.query_id = q.id
WHERE q.platform = ?
AND q.id NOT IN /* subtract the set of executions that are recent enough */
(
  SELECT l.query_id
  FROM labels l
  JOIN label_query_executions lqe
  ON lqe.label_id = l.id
  WHERE lqe.host_id = ? AND lqe.updated_at > ?
)`, host.Platform, host.ID, cutoff).Rows()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var id, query string
		err = rows.Scan(&id, &query)
		if err != nil {
			return results, nil
		}
		results[id] = query
	}

	return results, nil
}

func (orm gormDB) RecordLabelQueryExecutions(host *kolide.Host, results map[string]bool, t time.Time) error {
	if host == nil {
		return errors.New(
			"error recording host label query execution",
			"nil pointer passed to RecordLabelQueryExecutions",
		)
	}

	insert := new(bytes.Buffer)
	switch orm.Driver {
	case "mysql":
		insert.WriteString("INSERT ")
	case "sqlite3":
		insert.WriteString("REPLACE ")
	default:
		return errors.New(
			"Unknown DB driver",
			"Tried to use unknown DB driver in RecordLabelQueryExecutions: "+orm.Driver,
		)
	}

	insert.WriteString(
		"INTO label_query_executions (updated_at, matches, label_id, host_id) VALUES",
	)

	// Build up all the values and the query string
	vals := []interface{}{}
	for labelId, res := range results {
		insert.WriteString("(?,?,?,?),")
		vals = append(vals, t, res, labelId, host.ID)
	}

	queryString := insert.String()
	queryString = strings.TrimSuffix(queryString, ",")

	switch orm.Driver {
	case "mysql":
		queryString += `
ON DUPLICATE KEY UPDATE
updated_at = VALUES(updated_at),
matches = VALUES(matches)
`
	}

	if err := orm.DB.Debug().Exec(queryString, vals...).Error; err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (orm gormDB) NewPack(pack *kolide.Pack) error {
	if pack == nil {
		return errors.New(
			"error creating pack",
			"nil pointer passed to NewPack",
		)
	}
	return orm.DB.Create(pack).Error
}

func (orm gormDB) SavePack(pack *kolide.Pack) error {
	if pack == nil {
		return errors.New(
			"error saving pack",
			"nil pointer passed to SavePack",
		)
	}
	return orm.DB.Save(pack).Error
}

func (orm gormDB) DeletePack(pack *kolide.Pack) error {
	if pack == nil {
		return errors.New(
			"error deleting pack",
			"nil pointer passed to DeletePack",
		)
	}
	err := orm.DB.Delete(pack).Error
	if err != nil {
		return err
	}

	err = orm.DB.Where("pack_id = ?", pack.ID).Delete(&kolide.PackQuery{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (orm gormDB) Pack(id uint) (*kolide.Pack, error) {
	pack := &kolide.Pack{
		ID: id,
	}
	err := orm.DB.Where(pack).First(pack).Error
	if err != nil {
		return nil, err
	}
	return pack, nil
}

func (orm gormDB) Packs() ([]*kolide.Pack, error) {
	var packs []*kolide.Pack
	err := orm.DB.Find(&packs).Error
	return packs, err
}

func (orm gormDB) AddQueryToPack(query *kolide.Query, pack *kolide.Pack) error {
	if query == nil || pack == nil {
		return errors.New(
			"error adding query from pack",
			"nil pointer passed to AddQueryToPack",
		)
	}
	pq := &kolide.PackQuery{
		QueryID: query.ID,
		PackID:  pack.ID,
	}
	return orm.DB.Create(pq).Error
}

func (orm gormDB) GetQueriesInPack(pack *kolide.Pack) ([]*kolide.Query, error) {
	var queries []*kolide.Query
	if pack == nil {
		return nil, errors.New(
			"error getting queries in pack",
			"nil pointer passed to GetQueriesInPack",
		)
	}

	rows, err := orm.DB.Raw(`
SELECT
  q.id,
  q.created_at,
  q.updated_at,
  q.name,
  q.query,
  q.interval,
  q.snapshot,
  q.differential,
  q.platform,
  q.version
FROM 
  queries q 
JOIN 
  pack_queries pq 
ON 
  pq.query_id = q.id 
AND 
  pq.pack_id = ?;
`, pack.ID).Rows()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.DatabaseError(err)
	}
	defer rows.Close()

	for rows.Next() {
		query := new(kolide.Query)
		err = rows.Scan(
			&query.ID,
			&query.CreatedAt,
			&query.UpdatedAt,
			&query.Name,
			&query.Query,
			&query.Interval,
			&query.Snapshot,
			&query.Differential,
			&query.Platform,
			&query.Version,
		)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}

	return queries, nil
}

func (orm gormDB) RemoveQueryFromPack(query *kolide.Query, pack *kolide.Pack) error {
	if query == nil || pack == nil {
		return errors.New(
			"error removing query from pack",
			"nil pointer passed to RemoveQueryFromPack",
		)
	}
	pq := &kolide.PackQuery{
		QueryID: query.ID,
		PackID:  pack.ID,
	}
	return orm.DB.Where(pq).Delete(pq).Error
}
