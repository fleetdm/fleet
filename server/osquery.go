package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

	err = db.MarkHostSeen(&kolide.Host{NodeKey: body.NodeKey}, time.Now())

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

////////////////////////////////////////////////////////////////////////////////
// Query Management API Endpoints
////////////////////////////////////////////////////////////////////////////////

// swagger:response GetAllQueriesResponseBody
type GetAllQueriesResponseBody struct {
	Queries []*kolide.Query `json:"queries"`
}

// swagger:route GET /api/v1/kolide/queries
//
// Get information about all queries
//
// Using this API will allow the requester to inspect and get info on all
// queries that have been saved within a given instance of the kolide
// application
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetAllQueriesResponseBody
func GetAllQueries(c *gin.Context) {
	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	queries, err := ds.Queries()
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetAllQueriesResponseBody{
		Queries: queries,
	})
}

// swagger:response GetQueryResponseBody
type GetQueryResponseBody struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Interval     uint   `json:"interval"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
}

// swagger:route GET /api/v1/kolide/query/:id
//
// Get information about a query
//
// Using this API will allow the requester to inspect and get info on queries
// that have been saved within a given instance of the kolide application
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetQueryResponseBody
func GetQuery(c *gin.Context) {
	id, err := ParseAndValidateUrlID(c, "id")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	query, err := ds.Query(id)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetQueryResponseBody{
		ID:           query.ID,
		Name:         query.Name,
		Query:        query.Query,
		Interval:     query.Interval,
		Snapshot:     query.Snapshot,
		Differential: query.Differential,
		Platform:     query.Platform,
		Version:      query.Version,
	})
}

// swagger:parameters CreateQuery
type CreateQueryRequestBody struct {
	Name         string `json:"name" validate:"required"`
	Query        string `json:"query" validate:"required"`
	Interval     uint   `json:"interval"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
}

// swagger:route POST /api/v1/kolide/queries
//
// Create a new query
//
// Using this API will allow the requester to create a new query
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetQueryResponseBody
func CreateQuery(c *gin.Context) {
	var body CreateQueryRequestBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	query := &kolide.Query{
		Name:         body.Name,
		Query:        body.Query,
		Interval:     body.Interval,
		Snapshot:     body.Snapshot,
		Differential: body.Differential,
		Platform:     body.Platform,
		Version:      body.Version,
	}

	ds := GetDB(c)
	err = ds.NewQuery(query)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetQueryResponseBody{
		ID:           query.ID,
		Name:         query.Name,
		Query:        query.Query,
		Interval:     query.Interval,
		Snapshot:     query.Snapshot,
		Differential: query.Differential,
		Platform:     query.Platform,
		Version:      query.Version,
	})
}

// swagger:parameters ModifyQuery
type ModifyQueryRequestBody struct {
	Name         *string `json:"name"`
	Query        *string `json:"query"`
	Interval     *uint   `json:"interval"`
	Snapshot     *bool   `json:"snapshot"`
	Differential *bool   `json:"differential"`
	Platform     *string `json:"platform"`
	Version      *string `json:"version"`
}

// swagger:route PATCH /api/v1/kolide/queries/:id
//
// Modify a query
//
// Using this API will allow the requester to modify the parameters and
// attributes of a query
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetQueryResponseBody
func ModifyQuery(c *gin.Context) {
	var body ModifyQueryRequestBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}
	id, err := ParseAndValidateUrlID(c, "id")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	query, err := ds.Query(id)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	if body.Name != nil {
		query.Name = *body.Name
	}

	if body.Query != nil {
		query.Query = *body.Query
	}

	if body.Interval != nil {
		query.Interval = *body.Interval
	}

	if body.Snapshot != nil {
		query.Snapshot = *body.Snapshot
	}

	if body.Differential != nil {
		query.Differential = *body.Differential
	}

	if body.Platform != nil {
		query.Platform = *body.Platform
	}

	if body.Version != nil {
		query.Version = *body.Version
	}

	err = ds.SaveQuery(query)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetQueryResponseBody{
		ID:           query.ID,
		Name:         query.Name,
		Query:        query.Query,
		Interval:     query.Interval,
		Snapshot:     query.Snapshot,
		Differential: query.Differential,
		Platform:     query.Platform,
		Version:      query.Version,
	})
}

// swagger:route DELETE /api/v1/kolide/queries/:id
//
// Delete a query
//
// Using this API will allow the requester to delete a query
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: nil
func DeleteQuery(c *gin.Context) {
	id, err := ParseAndValidateUrlID(c, "id")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	query, err := ds.Query(id)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	err = ds.DeleteQuery(query)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

////////////////////////////////////////////////////////////////////////////////
// Pack Management API Endpoints
////////////////////////////////////////////////////////////////////////////////

// swagger:response GetAllPacksResponseBody
type GetAllPacksResponseBody struct {
	Packs []uint `json:"packs"`
}

// swagger:route GET /api/v1/kolide/packs
//
// Get information about all pack
//
// Using this API will allow the requester to get the IDs of all of the packs
// that have been created within a given instance of the kolide application
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetAllPacksResponseBody
func GetAllPacks(c *gin.Context) {
	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	packs, err := ds.Packs()
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	var packIDs []uint
	for _, pack := range packs {
		packIDs = append(packIDs, pack.ID)
	}

	c.JSON(http.StatusOK, GetAllPacksResponseBody{
		Packs: packIDs,
	})
}

// swagger:response GetPackResponseBody
type GetPackResponseBody struct {
	ID       uint                   `json:"id"`
	Name     string                 `json:"name"`
	Platform string                 `json:"platform"`
	Queries  []GetQueryResponseBody `json:"queries"`
}

// swagger:route GET /api/v1/kolide/pack/:id
//
// Get information about a pack
//
// Using this API will allow the requester to inspect and get info on packs
// that have been created within a given instance of the kolide application
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetPackResponseBody
func GetPack(c *gin.Context) {
	id, err := ParseAndValidateUrlID(c, "id")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	pack, err := ds.Pack(id)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	queries, err := ds.GetQueriesInPack(pack)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	var queriesResponse []GetQueryResponseBody
	for _, query := range queries {
		queriesResponse = append(queriesResponse, GetQueryResponseBody{
			ID:           query.ID,
			Name:         query.Name,
			Query:        query.Query,
			Interval:     query.Interval,
			Snapshot:     query.Snapshot,
			Differential: query.Differential,
			Platform:     query.Platform,
			Version:      query.Version,
		})
	}

	c.JSON(http.StatusOK, GetPackResponseBody{
		ID:       pack.ID,
		Name:     pack.Name,
		Platform: pack.Platform,
		Queries:  queriesResponse,
	})
}

// swagger:parameters CreatePack
type CreatePackRequestBody struct {
	Name     string `json:"name" validate:"required"`
	Platform string `json:"platform"`
}

// swagger:route PUT /api/v1/kolide/pack
//
// Create a new pack
//
// Using this API will allow the requester to create a new pack
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetPackResponseBody
func CreatePack(c *gin.Context) {
	var body CreatePackRequestBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	pack := &kolide.Pack{
		Name:     body.Name,
		Platform: body.Platform,
	}
	err = ds.NewPack(pack)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetPackResponseBody{
		ID:       pack.ID,
		Name:     pack.Name,
		Platform: pack.Platform,
	})
}

// swagger:parameters ModifyPack
type ModifyPackRequestBody struct {
	Name     *string `json:"name"`
	Platform *string `json:"platform"`
}

// swagger:route PATCH /api/v1/kolide/pack
//
// Modify a pack
//
// Using this API will allow the requester to modify the parameters and
// attributes of a pack
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetPackResponseBody
func ModifyPack(c *gin.Context) {
	var body ModifyPackRequestBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}
	id, err := ParseAndValidateUrlID(c, "id")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	pack, err := ds.Pack(uint(id))
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	if body.Name != nil {
		pack.Name = *body.Name
	}

	if body.Platform != nil {
		pack.Platform = *body.Platform
	}

	err = ds.SavePack(pack)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetPackResponseBody{
		ID:       pack.ID,
		Name:     pack.Name,
		Platform: pack.Platform,
	})
}

// swagger:route DELETE /api/v1/kolide/pack
//
// Delete a pack
//
// Using this API will allow the requester to delete a pack
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetPackResponseBody
func DeletePack(c *gin.Context) {
	id, err := ParseAndValidateUrlID(c, "id")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	pack, err := ds.Pack(uint(id))
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	err = ds.DeletePack(pack)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// swagger:route PUT /api/v1/kolide/packs/:pid/queries/:qid
//
// Add a query to a pack
//
// Using this API will allow the requester to add an existing query to an
// existing pack
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: nil
func AddQueryToPack(c *gin.Context) {
	packID, err := ParseAndValidateUrlID(c, "pid")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}
	queryID, err := ParseAndValidateUrlID(c, "qid")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	pack, err := ds.Pack(packID)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	query, err := ds.Query(queryID)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	err = ds.AddQueryToPack(query, pack)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// swagger:route DELETE /api/v1/kolide/packs/:pid/queries/:qid
//
// Delete a query from a pack
//
// Using this API will allow the requester to delete an existing query from an
// existing pack
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetPackResponseBody
func DeleteQueryFromPack(c *gin.Context) {
	packID, err := ParseAndValidateUrlID(c, "pid")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}
	queryID, err := ParseAndValidateUrlID(c, "qid")
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	ds := GetDB(c)
	pack, err := ds.Pack(uint(packID))
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	query, err := ds.Query(uint(queryID))
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	err = ds.RemoveQueryFromPack(query, pack)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
