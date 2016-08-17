package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/errors"
	"github.com/spf13/viper"
)

// The "output plugin" interface for osquery result logs. The implementer of
// this interface can do whatever processing they would like with the log,
// returning the appropriate error status
type OsqueryResultHandler interface {
	HandleResultLog(log OsqueryResultLog, nodeKey string) error
}

// The "output plugin" interface for osquery status logs. The implementer of
// this interface can do whatever processing they would like with the log,
// returning the appropriate error status
type OsqueryStatusHandler interface {
	HandleStatusLog(log OsqueryStatusLog, nodeKey string) error
}

// This struct is used for injecting dependencies for osquery TLS processing.
// It can be configured in a `main` function to bind the appropriate handlers
// and it's methods can be attached to routes.
type OsqueryHandler struct {
	ResultHandler OsqueryResultHandler
	StatusHandler OsqueryStatusHandler
}

// Basic implementation of the `OsqueryResultHandler` and
// `OsqueryStatusHandler` interfaces. It will write the logs to the io.Writer
// provided in Writer.
type OsqueryLogWriter struct {
	Writer io.Writer
}

func (w *OsqueryLogWriter) HandleStatusLog(log OsqueryStatusLog, nodeKey string) error {
	err := json.NewEncoder(w.Writer).Encode(log)
	if err != nil {
		return errors.NewFromError(err, http.StatusInternalServerError, "error writing result log")
	}
	return nil
}

func (w *OsqueryLogWriter) HandleResultLog(log OsqueryResultLog, nodeKey string) error {
	err := json.NewEncoder(w.Writer).Encode(log)
	if err != nil {
		return errors.NewFromError(err, http.StatusInternalServerError, "error writing status log")
	}
	return nil
}

type ScheduledQuery struct {
	ID           uint `gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Name         string `gorm:"not null"`
	QueryID      int
	Query        Query
	Interval     uint `gorm:"not null"`
	Snapshot     bool
	Differential bool
	Platform     string
	PackID       uint
}

type Query struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Query     string   `gorm:"not null"`
	Targets   []Target `gorm:"many2many:query_targets"`
}

type TargetType int

const (
	TargetLabel TargetType = iota
	TargetHost  TargetType = iota
)

type Target struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      TargetType
	QueryID   uint
	TargetID  uint
}

type DistributedQueryStatus int

const (
	QueryRunning  DistributedQueryStatus = iota
	QueryComplete DistributedQueryStatus = iota
	QueryError    DistributedQueryStatus = iota
)

type DistributedQuery struct {
	ID          uint `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Query       Query
	MaxDuration time.Duration
	Status      DistributedQueryStatus
	UserID      uint
}

type DistributedQueryExecutionStatus int

const (
	ExecutionWaiting   DistributedQueryExecutionStatus = iota
	ExecutionRequested DistributedQueryExecutionStatus = iota
	ExecutionSucceeded DistributedQueryExecutionStatus = iota
	ExecutionFailed    DistributedQueryExecutionStatus = iota
)

type DistributedQueryExecution struct {
	HostID             uint
	DistributedQueryID uint
	Status             DistributedQueryExecutionStatus
	Error              string `gorm:"size:1024"`
	ExecutionDuration  time.Duration
}

type Pack struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Name             string `gorm:"not null;unique_index:idx_pack_unique_name"`
	Platform         string
	Queries          []ScheduledQuery
	DiscoveryQueries []DiscoveryQuery
}

type DiscoveryQuery struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Query     string `gorm:"size:1024" gorm:"not null"`
}

type Host struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	NodeKey   string `gorm:"unique_index:idx_host_unique_nodekey"`
	HostName  string
	UUID      string `gorm:"unique_index:idx_host_unique_uuid"`
	IPAddress string
	Platform  string
	Labels    []*Label `gorm:"many2many:host_labels;"`
}

type Label struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"not null;unique_index:idx_label_unique_name"`
	Query     string
	Hosts     []Host
}

type Option struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Key       string `gorm:"not null;unique_index:idx_option_unique_key"`
	Value     string `gorm:"not null"`
	Platform  string
}

type DecoratorType int

const (
	DecoratorLoad     DecoratorType = iota
	DecoratorAlways   DecoratorType = iota
	DecoratorInterval DecoratorType = iota
)

type Decorator struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      DecoratorType `gorm:"not null"`
	Interval  int
	Query     string
}

type OsqueryEnrollPostBody struct {
	EnrollSecret   string `json:"enroll_secret" validate:"required"`
	HostIdentifier string `json:"host_identifier" validate:"required"`
}

type OsqueryConfigPostBody struct {
	NodeKey string `json:"node_key" validate:"required"`
}

type OsqueryLogPostBody struct {
	NodeKey string           `json:"node_key" validate:"required"`
	LogType string           `json:"log_type" validate:"required"`
	Data    *json.RawMessage `json:"data" validate:"required"`
}

type OsqueryResultLog struct {
	Name           string            `json:"name" validate:"required"`
	HostIdentifier string            `json:"hostIdentifier" validate:"required"`
	UnixTime       string            `json:"unixTime" validate:"required"`
	CalendarTime   string            `json:"calendarTime" validate:"required"`
	Columns        map[string]string `json:"columns"`
	Action         string            `json:"action" validate:"required"`
}

type OsqueryStatusLog struct {
	Severity    string            `json:"severity" validate:"required"`
	Filename    string            `json:"filename" validate:"required"`
	Line        string            `json:"line" validate:"required"`
	Message     string            `json:"message" validate:"required"`
	Version     string            `json:"version" validate:"required"`
	Decorations map[string]string `json:"decorations"`
}

type OsqueryDistributedReadPostBody struct {
	NodeKey string `json:"node_key" validate:"required"`
}

type OsqueryDistributedWritePostBody struct {
	NodeKey string                         `json:"node_key" validate:"required"`
	Queries map[string][]map[string]string `json:"queries" validate:"required"`
}

// Generate a node key using NodeKeySize random bytes Base64 encoded
func newNodeKey() (string, error) {
	return generateRandomText(viper.GetInt("osquery.node_key_size"))
}

// Enroll a host. Even if this is an existing host, a new node key should be
// generated and saved to the DB.
func EnrollHost(db *gorm.DB, uuid, hostName, ipAddress, platform string) (*Host, error) {
	host := Host{UUID: uuid}
	err := db.Where(&host).First(&host).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			// Create new Host
			host = Host{
				UUID:      uuid,
				HostName:  hostName,
				IPAddress: ipAddress,
				Platform:  platform,
			}

		default:
			return nil, err
		}
	}

	// Generate a new key each enrollment
	host.NodeKey, err = newNodeKey()
	if err != nil {
		return nil, err
	}

	// Update these fields if provided
	if hostName != "" {
		host.HostName = hostName
	}
	if ipAddress != "" {
		host.IPAddress = ipAddress
	}
	if platform != "" {
		host.Platform = platform
	}

	if err := db.Save(&host).Error; err != nil {
		return nil, err
	}

	return &host, nil
}

func OsqueryEnroll(c *gin.Context) {
	var body OsqueryEnrollPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	if body.EnrollSecret != viper.GetString("osquery.enroll_secret") {
		errors.ReturnOsqueryError(
			c,
			errors.NewWithStatus(http.StatusUnauthorized,
				"Node key invalid",
				fmt.Sprintf("Invalid node secret provided: %s", body.EnrollSecret),
			))
		return

	}

	db := GetDB(c)

	host, err := EnrollHost(db, body.HostIdentifier, "", "", "")
	if err != nil {
		errors.ReturnOsqueryError(c, errors.DatabaseError(err))
		return
	}

	logrus.Debugf("New host created: %+v", host)

	c.JSON(http.StatusOK,
		gin.H{
			"node_key":     host.NodeKey,
			"node_invalid": false,
		})
}

func OsqueryConfig(c *gin.Context) {
	var body OsqueryConfigPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}
	logrus.Debugf("OsqueryConfig: %s", body.NodeKey)

	c.JSON(http.StatusOK,
		gin.H{
			"schedule": map[string]map[string]interface{}{
				"time": {
					"query":    "select * from time;",
					"interval": 1,
				},
			},
			"node_invalid": false,
		})
}

// Authenticate a (post-enrollment) TLS request from osqueryd. To do this we
// verify that the provided node key is valid.
func authenticateRequest(db *gorm.DB, nodeKey string) error {
	host := Host{NodeKey: nodeKey}
	err := db.Where(&host).First(&host).Error
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
			return e
		default:
			return errors.DatabaseError(err)
		}
	}

	return nil
}

// Unmarshal the status logs before sending them to the status log handler
func (h *OsqueryHandler) handleStatusLogs(db *gorm.DB, data *json.RawMessage, nodeKey string) error {
	var statuses []OsqueryStatusLog
	if err := json.Unmarshal(*data, &statuses); err != nil {
		return errors.NewFromError(err, http.StatusBadRequest, "JSON parse error")
	}
	// Perhaps we should validate the unmarshalled status log

	for _, status := range statuses {
		if err := h.StatusHandler.HandleStatusLog(status, nodeKey); err != nil {
			return err
		}
		logrus.Debugf("Osquery status: %+v", status)
	}

	return nil
}

// Unmarshal the result logs before sending them to the result log handler
func (h *OsqueryHandler) handleResultLogs(db *gorm.DB, data *json.RawMessage, nodeKey string) error {
	var results []OsqueryResultLog
	if err := json.Unmarshal(*data, &results); err != nil {
		return errors.NewFromError(err, http.StatusBadRequest, "JSON parse error")
	}
	// Perhaps we should validate the unmarshalled result log

	for _, result := range results {
		if err := h.ResultHandler.HandleResultLog(result, nodeKey); err != nil {
			return err
		}
		logrus.Debugf("Osquery result: %+v", result)
	}

	return nil
}

// Set the update time for the provided host to indicate that it has
// successfully checked in.
func updateLastSeen(db *gorm.DB, host *Host) error {
	updateTime := time.Now()
	err := db.Exec("UPDATE hosts SET updated_at=? WHERE node_key=?", updateTime, host.NodeKey).Error
	if err != nil {
		return errors.DatabaseError(err)
	}
	host.UpdatedAt = updateTime
	return nil
}

// Endpoint used by the osqueryd TLS logger plugin
func (h *OsqueryHandler) OsqueryLog(c *gin.Context) {
	var body OsqueryLogPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	db := GetDB(c)

	err = authenticateRequest(db, body.NodeKey)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	switch body.LogType {
	case "status":
		err = h.handleStatusLogs(db, body.Data, body.NodeKey)

	case "result":
		err = h.handleResultLogs(db, body.Data, body.NodeKey)

	default:
		err = errors.NewWithStatus(
			errors.StatusUnprocessableEntity,
			"Unknown result type",
			fmt.Sprintf("Unknown result type: %s", body.LogType),
		)
	}

	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	err = updateLastSeen(db, &Host{NodeKey: body.NodeKey})

	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func OsqueryDistributedRead(c *gin.Context) {
	var body OsqueryDistributedReadPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}
	logrus.Debugf("OsqueryDistributedRead: %s", body.NodeKey)

	c.JSON(http.StatusOK,
		gin.H{
			"queries": map[string]string{
				"id1": "select * from osquery_info",
			},
			"node_invalid": false,
		})
}

func OsqueryDistributedWrite(c *gin.Context) {
	var body OsqueryDistributedWritePostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}
	logrus.Debugf("OsqueryDistributedWrite: %s", body.NodeKey)
	c.JSON(http.StatusOK,
		gin.H{
			"node_invalid": false,
		})
}
