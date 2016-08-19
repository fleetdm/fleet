package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
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

func OsqueryEnroll(c *gin.Context) {
	var body OsqueryEnrollPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	// TODO make config value explicit
	if body.EnrollSecret != viper.GetString("osquery.enroll_secret") {
		errors.ReturnOsqueryError(
			c,
			errors.NewWithStatus(http.StatusUnauthorized,
				"Node key invalid",
				fmt.Sprintf("Invalid node secret provided: %s", body.EnrollSecret),
			))
		return

	}

	// temporary, pass args explicitly as well
	db := GetDB(c)

	// TODO make config value explicit
	nodeKeySize := viper.GetInt("osquery.node_key_size")
	host, err := db.EnrollHost(body.HostIdentifier, "", "", "", nodeKeySize)
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

// Unmarshal the status logs before sending them to the status log handler
func (h *OsqueryHandler) handleStatusLogs(data *json.RawMessage, nodeKey string) error {
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
func (h *OsqueryHandler) handleResultLogs(data *json.RawMessage, nodeKey string) error {
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

// Endpoint used by the osqueryd TLS logger plugin
func (h *OsqueryHandler) OsqueryLog(c *gin.Context) {
	var body OsqueryLogPostBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	db := GetDB(c)

	_, err = db.AuthenticateHost(body.NodeKey)
	if err != nil {
		errors.ReturnOsqueryError(c, err)
		return
	}

	switch body.LogType {
	case "status":
		err = h.handleStatusLogs(body.Data, body.NodeKey)

	case "result":
		err = h.handleResultLogs(body.Data, body.NodeKey)

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

	err = db.UpdateLastSeen(&kolide.Host{NodeKey: body.NodeKey})

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
