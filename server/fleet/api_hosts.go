package fleet

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gocarina/gocsv"
)

type ListHostsRequest struct {
	Opts HostListOptions `url:"host_options"`
}

type ListHostsResponse struct {
	Hosts []HostResponse `json:"hosts"`
	// Software is populated with the software version corresponding to the
	// software_version_id (or software_id) filter if one is provided with the
	// request (and it exists in the database). It is nil otherwise and absent of
	// the JSON response payload.
	Software *Software `json:"software,omitempty"`
	// SoftwareTitle is populated with the title corresponding to the
	// software_title_id filter if one is provided with the request (and it
	// exists in the database). It is nil otherwise and absent of the JSON
	// response payload.
	SoftwareTitle *SoftwareTitle `json:"software_title,omitempty"`
	// MDMSolution is populated with the MDM solution corresponding to the mdm_id
	// filter if one is provided with the request (and it exists in the
	// database). It is nil otherwise and absent of the JSON response payload.
	MDMSolution *MDMSolution `json:"mobile_device_management_solution,omitempty"`
	// MunkiIssue is populated with the munki issue corresponding to the
	// munki_issue_id filter if one is provided with the request (and it exists
	// in the database). It is nil otherwise and absent of the JSON response
	// payload.
	MunkiIssue *MunkiIssue `json:"munki_issue,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r ListHostsResponse) Error() error { return r.Err }

type StreamHostsResponse struct {
	ListHostsResponse
	// HostResponseIterator is an iterator to stream hosts one by one.
	HostResponseIterator iter.Seq2[*HostResponse, error] `json:"-"`
	// MarshalJSON is an optional custom JSON marshaller for the response,
	// used for testing purposes only.
	MarshalJSON func(v any) ([]byte, error) `json:"-"`
}

func (r StreamHostsResponse) Error() error { return r.Err }

func (r StreamHostsResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	aliasRules := endpointer.ExtractAliasRules(ListHostsResponse{})
	w.Header().Set("Content-Type", "application/json")
	// If no iterator is provided, return a 500.
	if r.HostResponseIterator == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": "no host iterator provided"}`)
		return
	}

	// From here on we're committing to a "successful" response,
	// where the client will have to look for an `error` key
	// in the JSON to determine actual status.
	w.WriteHeader(http.StatusOK)

	// Create a no-op flush function in case the ResponseWriter doesn't implement http.Flusher.
	flush := func() {}
	if f, ok := w.(http.Flusher); ok {
		flush = f.Flush
	}

	// Use the default json marshaller unless a custom one is provided (for testing).
	marshalJson := json.Marshal
	if r.MarshalJSON != nil {
		marshalJson = r.MarshalJSON
	}

	// Create function for returning errors in the JSON response.
	marshalError := func(errString string) string {
		errData, err := json.Marshal(map[string]string{"error": errString})
		if err != nil {
			return `{"error": "unknown error"}`
		}
		return string(errData[1 : len(errData)-1])
	}

	// Start the JSON object.
	fmt.Fprint(w, `{`)
	firstKey := true

	t := reflect.TypeFor[ListHostsResponse]()
	v := reflect.ValueOf(r.ListHostsResponse)

	// The set of properties of ListHostsResponse to consider for output.
	fieldNames := []string{"Software", "SoftwareTitle", "MDMSolution", "MunkiIssue"}

	// Iterate over the non-host keys in the response and write them if they are non-nil.
	for i, fieldName := range fieldNames {
		// Get the JSON tag name for the field.
		fieldDef, _ := t.FieldByName(fieldName)
		tag := fieldDef.Tag.Get("json")
		parts := strings.Split(tag, ",")
		name := parts[0]

		// Get the actual value for the field.
		fieldValue := v.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			// Panic if the field is not found.
			// This indicates a programming error (we put something bad in the keys list).
			panic(fmt.Sprintf("field %s not found in ListHostsResponse", fieldName))
		}
		if !fieldValue.IsNil() {
			if i > 0 && !firstKey {
				fmt.Fprint(w, `,`)
			}
			data, err := marshalJson(fieldValue.Interface())
			if err != nil {
				// On error, write the error key and return.
				// Marshal the error as a JSON object without the surrounding braces,
				// in case the error string itself contains characters that would break
				// the JSON response.
				fmt.Fprint(w, marshalError(fmt.Sprintf("marshaling %s: %s", name, err.Error())))
				fmt.Fprint(w, `}`)
				return
			}
			// Output the key and value.
			fmt.Fprintf(w, `"%s":`, name)
			fmt.Fprint(w, string(data))
			flush()
			firstKey = false
		}
	}

	if !firstKey {
		fmt.Fprint(w, `,`)
	}

	// Start the hosts array.
	fmt.Fprint(w, `"hosts": [`)
	firstHost := true
	// Get hosts one at a time from the iterator and write them out.
	for hostResp, err := range r.HostResponseIterator {
		if err != nil {
			fmt.Fprint(w, `],`)
			fmt.Fprint(w, marshalError(fmt.Sprintf("getting host %s: ", err.Error())))
			fmt.Fprint(w, `}`)
			return
		}
		data, err := marshalJson(hostResp)
		if err != nil {
			fmt.Fprint(w, `],`)
			fmt.Fprint(w, marshalError(fmt.Sprintf("marshaling host response: %s", err.Error())))
			fmt.Fprint(w, `}`)
			return
		}
		data = endpointer.DuplicateJSONKeys(data, aliasRules, endpointer.DuplicateJSONKeysOpts{Compact: true})
		if !firstHost {
			fmt.Fprint(w, `,`)
		}
		fmt.Fprint(w, string(data))
		flush()
		firstHost = false
	}
	// Close the hosts array and the JSON object.
	fmt.Fprint(w, `]}`)
}

type DeleteHostsRequest struct {
	IDs []uint `json:"ids"`
	// Using a pointer to help determine whether an empty filter was passed, like: "filters":{}
	Filters *map[string]interface{} `json:"filters"`
}

type DeleteHostsResponse struct {
	Err        error `json:"error,omitempty"`
	StatusCode int   `json:"-"`
}

func (r DeleteHostsResponse) Error() error { return r.Err }

func (r DeleteHostsResponse) Status() int { return r.StatusCode }

type CountHostsRequest struct {
	Opts    HostListOptions `url:"host_options"`
	LabelID *uint                 `query:"label_id,optional"`
}

type CountHostsResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r CountHostsResponse) Error() error { return r.Err }

type SearchHostsRequest struct {
	// MatchQuery is the query SQL
	MatchQuery string `json:"query"`
	// QueryID is the ID of a saved query to run (used to determine if this is a
	// query that observers can run).
	QueryID *uint `json:"query_id" renameto:"report_id"`
	// ExcludedHostIDs is the list of IDs selected on the caller side
	// (e.g. the UI) that will be excluded from the returned payload.
	ExcludedHostIDs []uint `json:"excluded_host_ids"`
}

type SearchHostsResponse struct {
	Hosts []*HostResponse `json:"hosts"`
	Err   error                 `json:"error,omitempty"`
}

func (r SearchHostsResponse) Error() error { return r.Err }

// HostDetailResponse is the response struct that contains the full host information
// with the HostDetail details.
type HostDetailResponse struct {
	HostDetail
	Status      HostStatus   `json:"status"`
	DisplayText string       `json:"display_text"`
	DisplayName string       `json:"display_name"`
	Geolocation *GeoLocation `json:"geolocation,omitempty"`
}

type GetHostRequest struct {
	ID              uint `url:"id"`
	ExcludeSoftware bool `query:"exclude_software,optional"`
}

type GetHostResponse struct {
	Host *HostDetailResponse `json:"host"`
	Err  error               `json:"error,omitempty"`
}

func (r GetHostResponse) Error() error { return r.Err }

type GetHostSummaryRequest struct {
	TeamID       *uint   `query:"team_id,optional" renameto:"fleet_id"`
	Platform     *string `query:"platform,optional"`
	LowDiskSpace *int    `query:"low_disk_space,optional"`
}

type GetHostSummaryResponse struct {
	HostSummary
	Err error `json:"error,omitempty"`
}

func (r GetHostSummaryResponse) Error() error { return r.Err }

type HostByIdentifierRequest struct {
	Identifier      string `url:"identifier"`
	ExcludeSoftware bool   `query:"exclude_software,optional"`
}

type DeleteHostRequest struct {
	ID uint `url:"id"`
}

type DeleteHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteHostResponse) Error() error { return r.Err }

type AddHostsToTeamRequest struct {
	TeamID  *uint  `json:"team_id" renameto:"fleet_id"`
	HostIDs []uint `json:"hosts"`
}

type AddHostsToTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r AddHostsToTeamResponse) Error() error { return r.Err }

type AddHostsToTeamByFilterRequest struct {
	TeamID  *uint                   `json:"team_id" renameto:"fleet_id"`
	Filters *map[string]interface{} `json:"filters"`
}

type AddHostsToTeamByFilterResponse struct {
	Err error `json:"error,omitempty"`
}

func (r AddHostsToTeamByFilterResponse) Error() error { return r.Err }

type RefetchHostRequest struct {
	ID uint `url:"id"`
}

type RefetchHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r RefetchHostResponse) Error() error {
	return r.Err
}

type GetHostQueryReportRequest struct {
	ID      uint `url:"id"`
	QueryID uint `url:"report_id"`
}

type GetHostQueryReportResponse struct {
	QueryID       uint                          `json:"query_id" renameto:"report_id"`
	HostID        uint                          `json:"host_id"`
	HostName      string                        `json:"host_name"`
	LastFetched   *time.Time                    `json:"last_fetched"`
	ReportClipped bool                          `json:"report_clipped"`
	Results       []HostQueryReportResult `json:"results"`
	Err           error                         `json:"error,omitempty"`
}

func (r GetHostQueryReportResponse) Error() error { return r.Err }

type ListHostDeviceMappingRequest struct {
	ID uint `url:"id"`
}

type ListHostDeviceMappingResponse struct {
	HostID        uint                       `json:"host_id"`
	DeviceMapping []*HostDeviceMapping `json:"device_mapping"`
	Err           error                      `json:"error,omitempty"`
}

func (r ListHostDeviceMappingResponse) Error() error { return r.Err }

type PutHostDeviceMappingRequest struct {
	ID     uint   `url:"id"`
	Email  string `json:"email"`
	Source string `json:"source,omitempty"`
}

type PutHostDeviceMappingResponse struct {
	HostID        uint                       `json:"host_id"`
	DeviceMapping []*HostDeviceMapping `json:"device_mapping"`
	Err           error                      `json:"error,omitempty"`
}

func (r PutHostDeviceMappingResponse) Error() error { return r.Err }

type DeleteHostIDPRequest struct {
	HostID uint `url:"id"`
}

type DeleteHostIDPResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteHostIDPResponse) Error() error { return r.Err }

func (r DeleteHostIDPResponse) Status() int  { return http.StatusNoContent }

type GetHostMDMRequest struct {
	ID uint `url:"id"`
}

type GetHostMDMResponse struct {
	*HostMDM
	Err error `json:"error,omitempty"`
}

func (r GetHostMDMResponse) Error() error { return r.Err }

type GetHostMDMSummaryResponse struct {
	AggregatedMDMData
	Err error `json:"error,omitempty"`
}

func (r GetHostMDMSummaryResponse) Error() error { return r.Err }

type GetHostMDMSummaryRequest struct {
	TeamID   *uint  `query:"team_id,optional" renameto:"fleet_id"`
	Platform string `query:"platform,optional"`
}

type GetMacadminsDataRequest struct {
	ID uint `url:"id"`
}

type GetMacadminsDataResponse struct {
	Err       error                `json:"error,omitempty"`
	Macadmins *MacadminsData `json:"macadmins"`
}

func (r GetMacadminsDataResponse) Error() error { return r.Err }

type GetAggregatedMacadminsDataRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetAggregatedMacadminsDataResponse struct {
	Err       error                          `json:"error,omitempty"`
	Macadmins *AggregatedMacadminsData `json:"macadmins"`
}

func (r GetAggregatedMacadminsDataResponse) Error() error { return r.Err }

type HostsReportRequest struct {
	Opts    HostListOptions `url:"host_options"`
	LabelID *uint                 `query:"label_id,optional"`
	Format  string                `query:"format"`
	Columns string                `query:"columns,optional"`
}

type HostsReportResponse struct {
	Columns []string              `json:"-"` // used to control the generated csv, see the HijackRender method
	Hosts   []*HostResponse `json:"-"` // they get rendered explicitly, in csv
	Err     error                 `json:"error,omitempty"`
}

func (r HostsReportResponse) Error() error { return r.Err }

func (r HostsReportResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	// post-process the Device Mappings for CSV rendering
	for _, h := range r.Hosts {
		if h.DeviceMapping != nil {
			// return the list of emails, comma-separated, as part of that single CSV field
			var dms []struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(*h.DeviceMapping, &dms); err != nil {
				// log the error but keep going
				logging.WithErr(ctx, err)
				continue
			}

			var sb strings.Builder
			for i, dm := range dms {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(dm.Email)
			}
			h.CSVDeviceMapping = sb.String()
		}
	}

	var buf bytes.Buffer
	if err := gocsv.Marshal(r.Hosts, &buf); err != nil {
		logging.WithErr(ctx, err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"failed to generate CSV file"}`)
		return
	}

	returnAll := len(r.Columns) == 0

	var outRows [][]string
	if !returnAll {
		// read back the CSV to filter out any unwanted columns
		recs, err := csv.NewReader(&buf).ReadAll()
		if err != nil {
			logging.WithErr(ctx, err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, `{"error":"failed to generate CSV file"}`)
			return
		}

		if len(recs) > 0 {
			// map the header names to their field index
			hdrs := make(map[string]int, len(recs))
			for i, hdr := range recs[0] {
				hdrs[hdr] = i
			}

			outRows = make([][]string, len(recs))
			for i, rec := range recs {
				for _, col := range r.Columns {
					colIx, ok := hdrs[col]
					if !ok {
						// invalid column name - it would be nice to catch this in the
						// endpoint before processing the results, but it would require
						// duplicating the list of columns from the Host's struct tags to a
						// map and keep this in sync, for what is essentially a programmer
						// mistake that should be caught and corrected early.
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						w.WriteHeader(http.StatusBadRequest)
						_, _ = fmt.Fprintf(w, `{"error":"invalid column name: %q"}`, col)
						return
					}
					outRows[i] = append(outRows[i], rec[colIx])
				}
			}
		}
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="Hosts %s.csv"`, time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	var err error
	if returnAll {
		_, err = io.Copy(w, &buf)
	} else {
		err = csv.NewWriter(w).WriteAll(outRows)
	}
	if err != nil {
		logging.WithErr(ctx, err)
	}
}

type OsVersionsRequest struct {
	ListOptions
	TeamID             *uint   `query:"team_id,optional" renameto:"fleet_id"`
	Platform           *string `query:"platform,optional"`
	Name               *string `query:"os_name,optional"`
	Version            *string `query:"os_version,optional"`
	MaxVulnerabilities *int    `query:"max_vulnerabilities,optional"`
}

type OsVersionsResponse struct {
	Meta            *PaginationMetadata `json:"meta,omitempty"`
	Count           int                       `json:"count"`
	CountsUpdatedAt *time.Time                `json:"counts_updated_at"`
	OSVersions      []OSVersion         `json:"os_versions"`
	Err             error                     `json:"error,omitempty"`
}

func (r OsVersionsResponse) Error() error { return r.Err }

type GetOSVersionRequest struct {
	ID                 uint  `url:"id"`
	TeamID             *uint `query:"team_id,optional" renameto:"fleet_id"`
	MaxVulnerabilities *int  `query:"max_vulnerabilities,optional"`
}

type GetOSVersionResponse struct {
	CountsUpdatedAt *time.Time       `json:"counts_updated_at"`
	OSVersion       *OSVersion `json:"os_version"`
	Err             error            `json:"error,omitempty"`
}

func (r GetOSVersionResponse) Error() error { return r.Err }

type GetHostEncryptionKeyRequest struct {
	ID uint `url:"id"`
}

type GetHostEncryptionKeyResponse struct {
	Err           error                        `json:"error,omitempty"`
	EncryptionKey *HostDiskEncryptionKey `json:"encryption_key,omitempty"`
	HostID        uint                         `json:"host_id,omitempty"`
}

func (r GetHostEncryptionKeyResponse) Error() error { return r.Err }

type GetHostHealthRequest struct {
	ID uint `url:"id"`
}

type GetHostHealthResponse struct {
	Err        error             `json:"error,omitempty"`
	HostID     uint              `json:"host_id,omitempty"`
	HostHealth *HostHealth `json:"health,omitempty"`
}

func (r GetHostHealthResponse) Error() error { return r.Err }

type AddLabelsToHostRequest struct {
	ID     uint     `url:"id"`
	Labels []string `json:"labels"`
}

type AddLabelsToHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r AddLabelsToHostResponse) Error() error { return r.Err }

type RemoveLabelsFromHostRequest struct {
	ID     uint     `url:"id"`
	Labels []string `json:"labels"`
}

type RemoveLabelsFromHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r RemoveLabelsFromHostResponse) Error() error { return r.Err }

type GetHostSoftwareRequest struct {
	ID uint `url:"id"`
	HostSoftwareTitleListOptions
}

type GetHostSoftwareResponse struct {
	Software []*HostSoftwareWithInstaller `json:"software"`
	Count    int                                `json:"count"`
	Meta     *PaginationMetadata          `json:"meta,omitempty"`
	Err      error                              `json:"error,omitempty"`
}

func (r GetHostSoftwareResponse) Error() error { return r.Err }

var listHostCertificatesSortCols = map[string]bool{
	"common_name":     true,
	"not_valid_after": true,
}

type ListHostCertificatesRequest struct {
	ID uint `url:"id"`
	ListOptions
}

func (r *ListHostCertificatesRequest) ValidateRequest() error {
	if r.ListOptions.OrderKey != "" && !listHostCertificatesSortCols[r.ListOptions.OrderKey] {
		return &BadRequestError{Message: "invalid order key"}
	}
	return nil
}

type ListHostCertificatesResponse struct {
	Certificates []*HostCertificatePayload `json:"certificates"`
	Meta         *PaginationMetadata       `json:"meta,omitempty"`
	Count        uint                            `json:"count"`
	Err          error                           `json:"error,omitempty"`
}

func (r ListHostCertificatesResponse) Error() error { return r.Err }

