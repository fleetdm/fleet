package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/platform/http/multipartform"
	"github.com/gorilla/mux"
)

////////////////////////////////////////////////////////////////////////////////
// Run Script on a Host (async)
////////////////////////////////////////////////////////////////////////////////

type RunScriptRequest struct {
	HostID         uint   `json:"host_id"`
	ScriptID       *uint  `json:"script_id"`
	ScriptContents string `json:"script_contents"`
	ScriptName     string `json:"script_name"`
	TeamID         uint   `json:"team_id" renameto:"fleet_id"`
}

type RunScriptResponse struct {
	Err         error  `json:"error,omitempty"`
	HostID      uint   `json:"host_id,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
}

func (r RunScriptResponse) Error() error { return r.Err }
func (r RunScriptResponse) Status() int  { return http.StatusAccepted }

////////////////////////////////////////////////////////////////////////////////
// Run Script on a Host (sync)
////////////////////////////////////////////////////////////////////////////////

type RunScriptSyncRequest struct {
	HostID         uint   `json:"host_id"`
	ScriptID       *uint  `json:"script_id"`
	ScriptContents string `json:"script_contents"`
	ScriptName     string `json:"script_name"`
	TeamID         uint   `json:"team_id" renameto:"fleet_id"`
}

type RunScriptSyncResponse struct {
	Err error `json:"error,omitempty"`
	*HostScriptResult
	HostTimeout bool `json:"host_timeout"`
}

func (r RunScriptSyncResponse) Error() error { return r.Err }
func (r RunScriptSyncResponse) Status() int {
	if r.HostTimeout {
		// The more proper response for a timeout on the server would be: StatusGatewayTimeout = 504
		// However, as described in https://github.com/fleetdm/fleet/issues/15430 we will send:
		// StatusRequestTimeout = 408 // RFC 9110, 15.5.9
		// See: https://github.com/fleetdm/fleet/issues/15430#issuecomment-1847345617
		return http.StatusRequestTimeout
	}
	return http.StatusOK
}

////////////////////////////////////////////////////////////////////////////////
// Get script result for a host
////////////////////////////////////////////////////////////////////////////////

type GetScriptResultRequest struct {
	ExecutionID string `url:"execution_id"`
}

type GetScriptResultResponse struct {
	ScriptContents string    `json:"script_contents"`
	ScriptID       *uint     `json:"script_id"`
	ExitCode       *int64    `json:"exit_code"`
	Output         string    `json:"output"`
	Message        string    `json:"message"`
	HostName       string    `json:"hostname"`
	HostTimeout    bool      `json:"host_timeout"`
	HostID         uint      `json:"host_id"`
	ExecutionID    string    `json:"execution_id"`
	Runtime        int       `json:"runtime"`
	CreatedAt      time.Time `json:"created_at"`

	Err error `json:"error,omitempty"`
}

func (r GetScriptResultResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Create a (saved) script (via a multipart file upload)
////////////////////////////////////////////////////////////////////////////////

type CreateScriptRequest struct {
	TeamID *uint
	Script *multipart.FileHeader
}

func (CreateScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var decoded CreateScriptRequest

	err := multipartform.Parse(ctx, r, platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val := r.MultipartForm.Value["fleet_id"]
	if len(val) > 0 {
		fleetID, err := strconv.ParseUint(val[0], 10, 64)
		if err != nil {
			return nil, &BadRequestError{Message: fmt.Sprintf("failed to decode fleet_id in multipart form: %s", err.Error())}
		}
		fid := uint(fleetID) // nolint:gosec // ignore G115
		decoded.TeamID = &fid
	}

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

type CreateScriptResponse struct {
	Err      error `json:"error,omitempty"`
	ScriptID uint  `json:"script_id,omitempty"`
}

func (r CreateScriptResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Delete a (saved) script
////////////////////////////////////////////////////////////////////////////////

type DeleteScriptRequest struct {
	ScriptID uint `url:"script_id"`
}

type DeleteScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteScriptResponse) Error() error { return r.Err }
func (r DeleteScriptResponse) Status() int  { return http.StatusNoContent }

////////////////////////////////////////////////////////////////////////////////
// List (saved) scripts (paginated)
////////////////////////////////////////////////////////////////////////////////

type ListScriptsRequest struct {
	TeamID      *uint       `query:"team_id,optional" renameto:"fleet_id"`
	ListOptions ListOptions `url:"list_options"`
}

type ListScriptsResponse struct {
	Meta    *PaginationMetadata `json:"meta"`
	Scripts []*Script           `json:"scripts"`
	Err     error               `json:"error,omitempty"`
}

func (r ListScriptsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get/download a (saved) script
////////////////////////////////////////////////////////////////////////////////

type GetScriptRequest struct {
	ScriptID uint   `url:"script_id"`
	Alt      string `query:"alt,optional"`
}

type GetScriptResponse struct {
	*Script
	Err error `json:"error,omitempty"`
}

func (r GetScriptResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Update Script Contents
////////////////////////////////////////////////////////////////////////////////

type UpdateScriptRequest struct {
	Script   *multipart.FileHeader
	ScriptID uint
}

func (UpdateScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var decoded UpdateScriptRequest

	err := r.ParseMultipartForm(platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	vars := mux.Vars(r)
	scriptIDStr, ok := vars["script_id"]
	if !ok {
		return nil, &BadRequestError{Message: "missing script id"}
	}
	scriptID, err := strconv.ParseUint(scriptIDStr, 10, 64)
	if err != nil {
		return nil, &BadRequestError{Message: "invalid script id"}
	}
	// Check if scriptID exceeds the maximum value for uint, code linter
	if scriptID > uint64(^uint(0)) {
		return nil, &BadRequestError{Message: "script id out of bounds"}
	}

	decoded.ScriptID = uint(scriptID)

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

type UpdateScriptResponse struct {
	Err      error `json:"error,omitempty"`
	ScriptID uint  `json:"script_id,omitempty"`
}

func (r UpdateScriptResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Host Script Details
////////////////////////////////////////////////////////////////////////////////

type GetHostScriptDetailsRequest struct {
	HostID      uint        `url:"id"`
	ListOptions ListOptions `url:"list_options"`
}

type GetHostScriptDetailsResponse struct {
	Scripts []*HostScriptDetail `json:"scripts"`
	Meta    *PaginationMetadata `json:"meta"`
	Err     error               `json:"error,omitempty"`
}

func (r GetHostScriptDetailsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Batch Replace Scripts
////////////////////////////////////////////////////////////////////////////////

type BatchSetScriptsRequest struct {
	TeamID   *uint           `json:"-" query:"team_id,optional" renameto:"fleet_id"`
	TeamName *string         `json:"-" query:"team_name,optional" renameto:"fleet_name"`
	DryRun   bool            `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Scripts  []ScriptPayload `json:"scripts"`
}

type BatchSetScriptsResponse struct {
	Scripts []ScriptResponse `json:"scripts"`
	Err     error            `json:"error,omitempty"`
}

func (r BatchSetScriptsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Batch Script Execution Status / List / Summary / Host Results
////////////////////////////////////////////////////////////////////////////////

type BatchScriptExecutionStatusRequest struct {
	BatchExecutionID string `url:"batch_execution_id"`
}

type BatchScriptExecutionListRequest struct {
	TeamID  uint    `query:"team_id,required" renameto:"fleet_id"`
	Status  *string `query:"status,optional"`
	Page    *uint   `query:"page,optional"`
	PerPage *uint   `query:"per_page,optional"`
}

type BatchScriptExecutionStatusResponse struct {
	BatchActivity
	Err error `json:"error,omitempty"`
}

func (r BatchScriptExecutionStatusResponse) Error() error { return r.Err }

// TODO - remove these once we retire batch script summary endpoint and code.
type (
	BatchScriptExecutionSummaryRequest  BatchScriptExecutionStatusRequest
	BatchScriptExecutionSummaryResponse struct {
		ScriptID    uint      `json:"script_id" db:"script_id"`
		ScriptName  string    `json:"script_name" db:"script_name"`
		TeamID      *uint     `json:"team_id" db:"team_id" renameto:"fleet_id"`
		CreatedAt   time.Time `json:"created_at" db:"created_at"`
		NumTargeted *uint     `json:"targeted" db:"num_targeted"`
		NumPending  *uint     `json:"pending" db:"num_pending"`
		NumRan      *uint     `json:"ran" db:"num_ran"`
		NumErrored  *uint     `json:"errored" db:"num_errored"`
		NumCanceled *uint     `json:"canceled" db:"num_canceled"`
		Err         error     `json:"error,omitempty"`
	}
)

func (r BatchScriptExecutionSummaryResponse) Error() error { return r.Err }

type BatchScriptExecutionListResponse struct {
	BatchScriptExecutions []BatchActivity    `json:"batch_executions"`
	Count                 uint               `json:"count"`
	Err                   error              `json:"error,omitempty"`
	Meta                  PaginationMetadata `json:"meta"`
}

func (r BatchScriptExecutionListResponse) Error() error { return r.Err }

type BatchScriptExecutionHostResultsRequest struct {
	BatchExecutionID     string                     `url:"batch_execution_id"`
	BatchExecutionStatus BatchScriptExecutionStatus `query:"status"`
	ListOptions          ListOptions                `url:"list_options"`
}

type BatchScriptExecutionHostResultsResponse struct {
	Hosts []BatchScriptHost  `json:"hosts"`
	Count uint               `json:"count"`
	Err   error              `json:"error,omitempty"`
	Meta  PaginationMetadata `json:"meta"`
}

func (r BatchScriptExecutionHostResultsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Batch Script Cancel
////////////////////////////////////////////////////////////////////////////////

type BatchScriptCancelRequest struct {
	BatchExecutionID string `url:"batch_execution_id"`
}

type BatchScriptCancelResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BatchScriptCancelResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Bulk script execution
////////////////////////////////////////////////////////////////////////////////

type BatchScriptRunRequest struct {
	ScriptID  uint            `json:"script_id"`
	HostIDs   []uint          `json:"host_ids"`
	Filters   *map[string]any `json:"filters"`
	NotBefore *time.Time      `json:"not_before"`
}

type BatchScriptRunResponse struct {
	BatchExecutionID string `json:"batch_execution_id"`
	Err              error  `json:"error,omitempty"`
}

func (r BatchScriptRunResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Lock host
////////////////////////////////////////////////////////////////////////////////

type LockHostRequest struct {
	HostID  uint `url:"id"`
	ViewPin bool `query:"view_pin,optional"`
}

type LockHostResponse struct {
	Err           error               `json:"error,omitempty"`
	DeviceStatus  DeviceStatus        `json:"device_status,omitempty"`
	PendingAction PendingDeviceAction `json:"pending_action,omitempty"`
	UnlockPIN     string              `json:"unlock_pin,omitempty"`
}

func (r LockHostResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Unlock host
////////////////////////////////////////////////////////////////////////////////

type UnlockHostRequest struct {
	HostID uint `url:"id"`
}

type UnlockHostResponse struct {
	HostID        *uint               `json:"host_id,omitempty"`
	UnlockPIN     string              `json:"unlock_pin,omitempty"`
	DeviceStatus  DeviceStatus        `json:"device_status,omitempty"`
	PendingAction PendingDeviceAction `json:"pending_action,omitempty"`
	Err           error               `json:"error,omitempty"`
}

func (r UnlockHostResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Wipe host
////////////////////////////////////////////////////////////////////////////////

type WipeHostRequest struct {
	HostID   uint `url:"id"`
	Metadata *MDMWipeMetadata
}

func (req *WipeHostRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	if r == nil {
		return nil
	}

	decoder := json.NewDecoder(io.LimitReader(r, 100*1024))
	metadata := MDMWipeMetadata{}
	if err := decoder.Decode(&metadata); err != nil {
		if err == io.EOF {
			// OK ... body is optional
			return nil
		}
		return &BadRequestError{
			Message:     "failed to unmarshal request body",
			InternalErr: err,
		}
	}
	req.Metadata = &metadata

	return nil
}

type WipeHostResponse struct {
	Err           error               `json:"error,omitempty"`
	DeviceStatus  DeviceStatus        `json:"device_status,omitempty"`
	PendingAction PendingDeviceAction `json:"pending_action,omitempty"`
}

func (r WipeHostResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Download File Response
////////////////////////////////////////////////////////////////////////////////

// DownloadFileResponse is a generic response used by endpoints that return a
// file as an attachment (profiles, scripts, etc.).
type DownloadFileResponse struct {
	Err         error `json:"error,omitempty"`
	Filename    string
	Content     []byte
	ContentType string // optional, defaults to application/octet-stream
}

func (r DownloadFileResponse) Error() error { return r.Err }

func (r DownloadFileResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.Content)))
	if r.ContentType == "" {
		r.ContentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", r.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.Filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.Content); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}
