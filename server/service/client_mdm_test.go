package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareWindowsMDMCommand(t *testing.T) {
	c := &Client{}

	validXML := []byte(`
	<Exec>
		<CmdID>some-id</CmdID>
		<Item></Item>
	</Exec>
	`)

	invalidCmdXML := []byte(`
	<Add>
		<CmdID>some-id</Cmd>
		<Item></Item>
	</Add>
	`)

	noCmdIDXML := []byte(`
	<Exec>
		<Item></Item>
	</Exec>
	`)

	t.Run("Modifies valid CmdID", func(t *testing.T) {
		modified, err := c.prepareWindowsMDMCommand(validXML)
		assert.Nil(t, err)

		doc := etree.NewDocument()
		err = doc.ReadFromBytes(modified)
		assert.Nil(t, err)

		element := doc.FindElement("//CmdID")
		assert.NotNil(t, element)
		assert.NotEmpty(t, element.Text())
	})

	t.Run("Adds CmdID if missing", func(t *testing.T) {
		modified, err := c.prepareWindowsMDMCommand(noCmdIDXML)
		assert.Nil(t, err)

		doc := etree.NewDocument()
		err = doc.ReadFromBytes(modified)
		assert.Nil(t, err)

		element := doc.FindElement("//CmdID")
		assert.NotNil(t, element)
		assert.NotEmpty(t, element.Text())
	})

	t.Run("Returns error on invalid XML", func(t *testing.T) {
		_, err := c.prepareWindowsMDMCommand(invalidCmdXML)
		assert.NotNil(t, err)

		_, err = c.prepareWindowsMDMCommand([]byte("<Exec><Exec"))
		assert.NotNil(t, err)
	})
}

func TestUploadBootstrapPackageStaged(t *testing.T) {
	for _, staged := range []bool{true, false} {
		var putBody []byte
		var form map[string][]string
		var gotPackage []byte
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/latest/fleet/config":
				_ = json.NewEncoder(w).Encode(map[string]any{"staged_upload_available": staged})
			case "/api/latest/fleet/staged_upload":
				var req createStagedUploadRequest
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				assert.Equal(t, fleet.StagedUploadTargetBootstrapPackage, req.Target)
				assert.EqualValues(t, 3, req.FleetID)
				assert.EqualValues(t, 3, req.Size)
				_ = json.NewEncoder(w).Encode(map[string]any{"upload_id": "up1", "url": "http://" + r.Host + "/object-store"})
			case "/object-store":
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Empty(t, r.Header.Get("Authorization"))
				putBody, _ = io.ReadAll(r.Body)
			case "/api/latest/fleet/bootstrap":
				assert.NoError(t, r.ParseMultipartForm(1<<20)) //nolint:gosec // test server
				form = r.MultipartForm.Value
				if f, _, err := r.FormFile("package"); err == nil {
					gotPackage, _ = io.ReadAll(f)
				}
				_, _ = w.Write([]byte("{}"))
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		}))
		client, err := NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		client.SetToken("test-token")

		require.NoError(t, client.UploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Name: "b.pkg", TeamID: 3, Bytes: []byte("pkg")}, false))
		srv.Close()

		if staged {
			require.Equal(t, "pkg", string(putBody))
			require.Equal(t, []string{"up1"}, form["upload_id"])
			require.Equal(t, []string{"b.pkg"}, form["filename"])
			require.Nil(t, gotPackage)
		} else {
			require.Nil(t, putBody)
			require.Empty(t, form["upload_id"])
			require.Equal(t, "pkg", string(gotPackage))
		}
		require.Equal(t, []string{"3"}, form["fleet_id"])
	}
}

func TestUploadBootstrapPackageIfNeededStagesBeforeDelete(t *testing.T) {
	for _, putStatus := range []int{http.StatusOK, http.StatusForbidden} {
		var calls []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.Method+" "+r.URL.Path)
			switch r.URL.Path {
			case "/api/latest/fleet/mdm/bootstrap/3/metadata":
				_ = json.NewEncoder(w).Encode(map[string]any{"name": "old.pkg", "sha256": []byte("old")})
			case "/api/latest/fleet/config":
				_ = json.NewEncoder(w).Encode(map[string]any{"staged_upload_available": true})
			case "/api/latest/fleet/staged_upload":
				_ = json.NewEncoder(w).Encode(map[string]any{"upload_id": "up1", "url": "http://" + r.Host + "/object-store"})
			case "/object-store":
				w.WriteHeader(putStatus)
			default:
				_, _ = w.Write([]byte("{}"))
			}
		}))
		client, err := NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		client.SetToken("test-token")

		err = client.UploadBootstrapPackageIfNeeded(&fleet.MDMAppleBootstrapPackage{Name: "b.pkg", Bytes: []byte("pkg"), Sha256: []byte("new")}, 3, false)
		srv.Close()

		if putStatus == http.StatusOK {
			require.NoError(t, err)
			require.Equal(t, []string{
				"GET /api/latest/fleet/mdm/bootstrap/3/metadata",
				"GET /api/latest/fleet/config",
				"POST /api/latest/fleet/staged_upload",
				"PUT /object-store",
				"DELETE /api/latest/fleet/mdm/bootstrap/3",
				"POST /api/latest/fleet/bootstrap",
			}, calls)
		} else {
			require.Error(t, err)
			require.NotContains(t, calls, "DELETE /api/latest/fleet/mdm/bootstrap/3")
		}
	}
}

// eulaMetadataStub answers like a current server with no agreement, including the client's support probe.
func eulaMetadataStub(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("platform") == "probe" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message": "Validation Failed", "errors": [{"name": "platform", "reason": "must be \"darwin\" or \"windows\""}]}`))
		return
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"message": "Resource Not Found"}`))
}

func TestGitOpsWindowsEULAWaitsForWindowsMDMToSettle(t *testing.T) {
	var waits int
	settle := windowsMDMSettle
	windowsMDMSettle = func() { waits++ }
	t.Cleanup(func() { windowsMDMSettle = settle })

	mdPath := filepath.Join(t.TempDir(), "terms.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Terms\n"), 0o600))

	for _, tt := range []struct {
		name        string
		mdmOnBefore bool
		wantWaits   int
	}{
		{name: "turned on by this run", wantWaits: 1},
		{name: "already on", mdmOnBefore: true, wantWaits: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			waits = 0
			var uploads int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.URL.Path {
				case "GET /api/latest/fleet/setup_experience/eula/metadata":
					eulaMetadataStub(w, r)
				case "POST /api/latest/fleet/setup_experience/eula":
					uploads++
					_, _ = w.Write([]byte("{}"))
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer srv.Close()
			client, err := NewClient(srv.URL, true, "", "")
			require.NoError(t, err)
			client.SetToken("test-token")

			appConfig := &fleet.EnrichedAppConfig{}
			appConfig.License = &fleet.LicenseInfo{Tier: fleet.TierPremium}
			appConfig.MDM.WindowsEnabledAndConfigured = tt.mdmOnBefore
			assumptions := &fleet.TeamSpecsDryRunAssumptions{WindowsEnabledAndConfigured: optjson.SetBool(true)}

			require.NoError(t, client.doGitOpsWindowsEULA(mdPath, appConfig, assumptions, "", false, func(string, ...any) {}))
			require.Equal(t, tt.wantWaits, waits)
			require.Equal(t, 1, uploads)
		})
	}
}

// Servers from before Windows EULAs ignore the platform parameter and answer for the macOS EULA, whatever its
// state. The client must leave that EULA alone, and stay quiet when windows_eula isn't set.
func TestGitOpsWindowsEULAServerSupport(t *testing.T) {
	const metadataPath = "/api/latest/fleet/setup_experience/eula/metadata"
	macOSEULA := `{"name": "eula.pdf", "token": "mac-token", "created_at": "2026-10-01T00:00:00Z"}`
	mdmOff := `{"message": "` + fleet.MDMNotConfiguredMessage + `", "errors": [{"name": "base", "reason": "` + fleet.MDMNotConfiguredMessage + `"}]}`
	answer := func(status int, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}
	}

	mdPath := filepath.Join(t.TempDir(), "terms.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Terms\n"), 0o600))

	for _, tt := range []struct {
		name      string
		metadata  http.HandlerFunc
		supported bool
	}{
		{"current server", eulaMetadataStub, true},
		{"older server with a macOS EULA", answer(http.StatusOK, macOSEULA), false},
		{"older server without an EULA", answer(http.StatusNotFound, `{"message": "Resource Not Found"}`), false},
		{"older server with Apple MDM off", answer(http.StatusBadRequest, mdmOff), false},
	} {
		for _, path := range []string{mdPath, ""} {
			t.Run(fmt.Sprintf("%s, windows_eula set: %t", tt.name, path != ""), func(t *testing.T) {
				var requests []string
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, r.Method+" "+r.URL.RequestURI())
					switch r.Method + " " + r.URL.Path {
					case "GET " + metadataPath:
						tt.metadata(w, r)
					case "POST /api/latest/fleet/setup_experience/eula":
						// the server reads the platform from the form, never from the query
						assert.Equal(t, "windows", r.PostFormValue("platform"))
						assert.Empty(t, r.URL.Query().Get("platform"))
						_, _ = w.Write([]byte("{}"))
					default:
						t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					}
				}))
				defer srv.Close()
				client, err := NewClient(srv.URL, true, "", "")
				require.NoError(t, err)
				client.SetToken("test-token")

				appConfig := &fleet.EnrichedAppConfig{}
				appConfig.License = &fleet.LicenseInfo{Tier: fleet.TierPremium}
				appConfig.MDM.WindowsEnabledAndConfigured = true
				var logged strings.Builder
				logFn := func(format string, args ...any) { fmt.Fprintf(&logged, format, args...) }
				require.NoError(t, client.doGitOpsWindowsEULA(path, appConfig, nil, "", false, logFn))

				probe := "GET " + metadataPath + "?platform=probe"
				switch {
				case tt.supported && path != "":
					require.Equal(t, []string{probe, "GET " + metadataPath + "?platform=windows", "POST /api/latest/fleet/setup_experience/eula?dry_run=false"}, requests)
					require.Equal(t, "[+] applied Windows EULA\n", logged.String())
				case tt.supported:
					require.Equal(t, []string{probe, "GET " + metadataPath + "?platform=windows"}, requests)
					require.Equal(t, "[+] applied Windows EULA\n", logged.String())
				case path != "":
					require.Equal(t, []string{probe}, requests)
					require.Equal(t, "[!] skipping windows_eula: "+ErrWindowsEULANotSupported.Error()+"\n", logged.String())
				default:
					require.Equal(t, []string{probe}, requests)
					require.Empty(t, logged.String())
				}
			})
		}
	}
}

func TestEULAMacOSRequestsSendNoPlatform(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		_, _ = w.Write([]byte(`{"name": "eula.pdf", "token": "mac-token", "created_at": "2026-10-01T00:00:00Z"}`))
	}))
	defer srv.Close()
	client, err := NewClient(srv.URL, true, "", "")
	require.NoError(t, err)
	client.SetToken("test-token")

	meta, err := client.GetEULAMetadata()
	require.NoError(t, err)
	require.Equal(t, "mac-token", meta.Token)
	require.Equal(t, []string{"GET /api/latest/fleet/setup_experience/eula/metadata"}, requests)
}
