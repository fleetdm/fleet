package app

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/kolide/kolide-ose/errors"
)

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

//

type OsqueryEnrollPostBody struct {
	EnrollSecret string `json:"enroll_secret" validate:"required"`
}

type OsqueryConfigPostBody struct {
	NodeKey string `json:"node_key" validate:"required"`
}

type OsqueryLogPostBody struct {
	NodeKey string                   `json:"node_key" validate:"required"`
	LogType string                   `json:"log_type" validate:"required"`
	Data    []map[string]interface{} `json:"data" validate:"required"`
}

type OsqueryResultLog struct {
	Name           string            `json:"name"`
	HostIdentifier string            `json:"hostIdentifier"`
	UnixTime       string            `json:"unixTime"`
	CalendarTime   string            `json:"calendarTime"`
	Columns        map[string]string `json:"columns"`
	Action         string            `json:"action"`
}

type OsqueryStatusLog struct {
	Severity string `json:"severity"`
	Filename string `json:"filename"`
	Line     string `json:"line"`
	Message  string `json:"message"`
	Version  string `json:"version"`
}

type OsqueryDistributedReadPostBody struct {
	NodeKey string `json:"node_key" validate:"required"`
}

type OsqueryDistributedWritePostBody struct {
	NodeKey string                         `json:"node_key" validate:"required"`
	Queries map[string][]map[string]string `json:"queries" validate:"required"`
}

func OsqueryEnroll(c *gin.Context) {
	var body OsqueryEnrollPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}
	logrus.Debugf("OsqueryEnroll: %s", body.EnrollSecret)

	c.JSON(http.StatusOK,
		gin.H{
			"node_key":     "7",
			"node_invalid": false,
		})
}

func OsqueryConfig(c *gin.Context) {
	var body OsqueryConfigPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
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

func OsqueryLog(c *gin.Context) {
	var body OsqueryLogPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}
	logrus.Debugf("OsqueryLog: %s", body.LogType)

	if body.LogType == "status" {
		for _, data := range body.Data {
			var log OsqueryStatusLog

			severity, ok := data["severity"].(string)
			if ok {
				log.Severity = severity
			} else {
				logrus.Error("Error asserting the type of status log severity")
			}

			filename, ok := data["filename"].(string)
			if ok {
				log.Filename = filename
			} else {
				logrus.Error("Error asserting the type of status log filename")
			}

			line, ok := data["line"].(string)
			if ok {
				log.Line = line
			} else {
				logrus.Error("Error asserting the type of status log line")
			}

			message, ok := data["message"].(string)
			if ok {
				log.Message = message
			} else {
				logrus.Error("Error asserting the type of status log message")
			}

			version, ok := data["version"].(string)
			if ok {
				log.Version = version
			} else {
				logrus.Error("Error asserting the type of status log version")
			}

			logrus.WithFields(logrus.Fields{
				"node_key": body.NodeKey,
				"severity": log.Severity,
				"filename": log.Filename,
				"line":     log.Line,
				"version":  log.Version,
			}).Info(log.Message)
		}
	} else if body.LogType == "result" {
		// TODO: handle all of the different kinds of results logs
	}

	c.JSON(http.StatusOK,
		gin.H{
			"node_invalid": false,
		})
}

func OsqueryDistributedRead(c *gin.Context) {
	var body OsqueryDistributedReadPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
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
		errors.ReturnError(c, err)
		return
	}
	logrus.Debugf("OsqueryDistributedWrite: %s", body.NodeKey)
	c.JSON(http.StatusOK,
		gin.H{
			"node_invalid": false,
		})
}
