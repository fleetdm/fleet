package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gocarina/gocsv"
)

// ///////////////////////////////////////////////////////////////////////////////
// Get policy status
// ///////////////////////////////////////////////////////////////////////////////

func getPolicyStatusEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	if !license.IsPremium(ctx) {
		return nil, fleet.ErrMissingLicense
	}

	req := request.(*fleet.GetPolicyStatusRequest)
	policy, err := svc.GetPolicyByID(ctx, req.PolicyID)
	if err != nil {
		return &fleet.GetPolicyStatusResponse{Err: err}, nil
	}

	return svc.GetPolicyStatus(ctx, policy, *req)
}

func (svc Service) GetPolicyStatus(ctx context.Context, policy *fleet.Policy, req fleet.GetPolicyStatusRequest) (*fleet.GetPolicyStatusResponse, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	switch req.RunStatus {
	case "", "policy_failed", "automation_failed":
		// valid
	default:
		return nil, fleet.NewInvalidArgumentError("run_status", `must be one of "policy_failed", "automation_failed"`)
	}

	// Default to newest-first so pages are stable when the caller omits order_key.
	if req.ListOptions.OrderKey == "" {
		req.ListOptions.OrderKey = "created_at"
		req.ListOptions.OrderDirection = fleet.OrderDescending
	}

	// IncludeObserver:true so team observers can read policy status for teams
	// they observe; the calling user has already been authorized to read the
	// policy itself via GetPolicyByID.
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	runs, count, meta, err := svc.ds.GetPolicyStatus(ctx, policy.ID, filter, req)
	if err != nil {
		return nil, err
	}

	return &fleet.GetPolicyStatusResponse{
		Runs:  runs,
		Count: count,
		Meta:  meta,
	}, nil
}

// ///////////////////////////////////////////////////////////////////////////////
// Export policy status (CSV)
// ///////////////////////////////////////////////////////////////////////////////

type exportPolicyStatusRequest struct {
	PolicyID             uint   `url:"policy_id"`
	HostNameQuery        string `query:"hostname,optional"`
	AutomationErrorQuery string `query:"automation_error,optional"`
	RunStatus            string `query:"run_status,optional"`
}

type policyStatusCSVRow struct {
	HostID              uint      `csv:"host_id"`
	HostName            string    `csv:"host_name"`
	Status              string    `csv:"status"`
	ConsecutiveFailures uint      `csv:"consecutive_failures"`
	CreatedAt           time.Time `csv:"created_at"`
	AutomationType      string    `csv:"automation_type"`
	AutomationStatus    string    `csv:"automation_status"`
	AutomationError     string    `csv:"automation_error"`
}

type exportPolicyStatusResponse struct {
	Rows []policyStatusCSVRow
	Err  error `json:"error,omitempty"`
}

func (r exportPolicyStatusResponse) Error() error { return r.Err }

func (r exportPolicyStatusResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	// Marshal into a buffer first so that if encoding fails we can still
	// return an error response — once WriteHeader(200) is called we can
	// no longer change the status code.
	var buf bytes.Buffer
	if err := gocsv.Marshal(r.Rows, &buf); err != nil {
		logging.WithErr(ctx, err)
		encodeError(ctx, ctxerr.New(ctx, "failed to generate CSV file"), w)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="Policy Status %s.csv"`, time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, &buf); err != nil {
		logging.WithErr(ctx, err)
	}
}

func exportPolicyStatusEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	if !license.IsPremium(ctx) {
		return nil, fleet.ErrMissingLicense
	}
	req := request.(*exportPolicyStatusRequest)
	policy, err := svc.GetPolicyByID(ctx, req.PolicyID)
	if err != nil {
		return exportPolicyStatusResponse{Err: err}, nil
	}
	statusReq := fleet.GetPolicyStatusRequest{
		PolicyID:             req.PolicyID,
		HostNameQuery:        req.HostNameQuery,
		AutomationErrorQuery: req.AutomationErrorQuery,
		RunStatus:            req.RunStatus,
		// ListOptions zero-value → PerPage=0, which GetPerPage() maps to
		// fleet.DefaultPerPage (1,000,000). This is Fleet's "effectively unbounded"
		// convention used across all list endpoints — sufficient for real fleets.
	}
	resp, err := svc.GetPolicyStatus(ctx, policy, statusReq)
	if err != nil {
		return exportPolicyStatusResponse{Err: err}, nil
	}
	return exportPolicyStatusResponse{Rows: flattenPolicyStatusToCSV(resp.Runs)}, nil
}

func flattenPolicyStatusToCSV(runs []fleet.GetPolicyStatusPolicyRun) []policyStatusCSVRow {
	rows := make([]policyStatusCSVRow, 0, len(runs))
	for _, run := range runs {
		status := "passing"
		if !run.NewStatus {
			status = "failing"
		}
		base := policyStatusCSVRow{
			HostID:              run.HostID,
			HostName:            run.HostName,
			Status:              status,
			ConsecutiveFailures: run.ConsecutiveFailures,
			CreatedAt:           run.CreatedAt,
		}
		if len(run.AutomationExecutions) == 0 {
			rows = append(rows, base)
			continue
		}
		for _, a := range run.AutomationExecutions {
			row := base
			row.AutomationType = a.Type
			row.AutomationStatus = a.Status
			row.AutomationError = a.ErrorMessage
			rows = append(rows, row)
		}
	}
	return rows
}
