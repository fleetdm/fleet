package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/ejbca/ejbcatest"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep/sceptest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestBatchApplyCertificateAuthorities() {
	t := s.T()

	// TODO(hca): test each CA type activities once implemented

	// 	TODO(hca) test each CA type cannot configure without private key?

	// TODO(hca); add mechanisms for each CA type to check that external validation endpoints are called for
	// new/modified CAs and that they are not called if nothing changes?

	// TODO(hca): test free version disallows batch endpoint

	ndesSCEPServer := sceptest.NewTestSCEPServer(t)
	ndesAdminServer := sceptest.NewTestNDESAdminServer(t, "mscep_admin_password", http.StatusOK)
	dynamicChallengeServer := sceptest.NewTestDynamicChallengeServer(t)

	pathRegex := regexp.MustCompile(`^/mpki/api/v2/profile/([a-zA-Z0-9_-]+)$`)
	mockDigiCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		matches := pathRegex.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		profileID := matches[1]
		resp := map[string]string{
			"id":     profileID,
			"name":   "Test CA",
			"status": "Active",
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	t.Cleanup(mockDigiCertServer.Close)

	mockHydrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/cacerts" {
			w.Header().Set("Content-Type", "application/pkcs7-mime")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Imagine if there was actually CA cert data here..."))
			require.NoError(t, err)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	t.Cleanup(mockHydrantServer.Close)

	mockSCEPServer := scep_server.StartTestSCEPServer(t)
	t.Cleanup(mockSCEPServer.Close)

	// goodNDESSCEPCA is a base object for testing with a valid NDES SCEP CA. Copy it to override specific fields in tests.
	goodNDESSCEPCA := fleet.NDESSCEPProxyCA{
		URL:      ndesSCEPServer.URL + "/scep",
		AdminURL: ndesAdminServer.URL + "/mscep_admin/",
		Username: "user",
		Password: "password",
	}

	// goodDigiCertCA is a base object for testing with a valid DigiCert CA. Copy it to override specific fields in tests.
	goodDigiCertCA := fleet.DigiCertCA{
		Name:                          "VALID_DIGICERT_CA",
		URL:                           mockDigiCertServer.URL,
		APIToken:                      "token",
		ProfileID:                     "valid-profile-id",
		CertificateCommonName:         "common-name",
		CertificateUserPrincipalNames: []string{"user1@example.com"},
		CertificateSeatID:             "seat-id",
	}

	// goodCustomSCEPCA is a base object for testing with a valid Custom SCEP CA. Copy it to override specific fields in tests.
	goodCustomSCEPCA := fleet.CustomSCEPProxyCA{
		Name:      "VALID_CUSTOM_SCEP",
		URL:       mockSCEPServer.URL + "/scep",
		Challenge: "challenge",
	}

	// goodHydrantCA is a base object for testing with a valid Hydrant CA. Copy it to override specific fields in tests.
	goodHydrantCA := fleet.HydrantCA{
		Name:         "VALID_HYDRANT",
		URL:          mockHydrantServer.URL, // TODO: implement a test server?
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	goodESTCA := fleet.ESTProxyCA{
		Name:     "VALID_EST",
		URL:      mockHydrantServer.URL,
		Username: "username",
		Password: "password",
	}

	// goodSmallstepCA is a base object for testing with a valid Smallstep SCEP CA. Copy it to override specific fields in tests.
	goodSmallstepCA := fleet.SmallstepSCEPProxyCA{
		Name:         "VALID_SMALLSTEP_SCEP",
		URL:          mockSCEPServer.URL + "/scep",
		ChallengeURL: dynamicChallengeServer.URL + "/challenge",
		Username:     "user",
		Password:     "password",
	}

	// goodGoogleCASCA is a base object for testing with a valid Google CAS CA. Copy it to override specific fields in tests.
	goodGoogleCASCA := fleet.GoogleCASCA{
		Name:       "VALID_GOOGLE_CAS",
		Project:    "my-project",
		Location:   "us-central1",
		CAPool:     "my-ca-pool",
		APIKeyJSON: `{"type":"service_account","client_email":"sa@my-project.iam.gserviceaccount.com","private_key":"key"}`,
	}
	s.googleCASService.VerifyCAPoolFunc = func(_ context.Context, _ fleet.GoogleCASCA) error { return nil }

	// newApplyRequest creates a new applyCertificateAuthoritiesSpecRequest. The given payload
	// should be one of fleet.DigiCertCA, fleet.CustomSCEPProxyCA, fleet.HydrantCA, or fleet.NDESSCEPProxyCA.
	newApplyRequest := func(p interface{}, dryRun bool) (batchApplyCertificateAuthoritiesRequest, error) {
		switch v := p.(type) {
		case fleet.CustomSCEPProxyCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					CustomScepProxy: []fleet.CustomSCEPProxyCA{v},
				},
				DryRun: dryRun,
			}, nil
		case fleet.HydrantCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					Hydrant: []fleet.HydrantCA{v},
				},
				DryRun: dryRun,
			}, nil
		case fleet.ESTProxyCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					EST: []fleet.ESTProxyCA{v},
				},
				DryRun: dryRun,
			}, nil
		case fleet.NDESSCEPProxyCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					NDESSCEP: &v,
				},
				DryRun: dryRun,
			}, nil
		case fleet.DigiCertCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert: []fleet.DigiCertCA{v},
				},
				DryRun: dryRun,
			}, nil
		case fleet.SmallstepSCEPProxyCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					Smallstep: []fleet.SmallstepSCEPProxyCA{v},
				},
				DryRun: dryRun,
			}, nil
		case fleet.GoogleCASCA:
			return batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					GoogleCAS: []fleet.GoogleCASCA{v},
				},
				DryRun: dryRun,
			}, nil
		default:
			return batchApplyCertificateAuthoritiesRequest{}, errors.New("invalid usage of newApplyRequest")
		}
	}

	// common invalid name test cases for DigiCert, Custom SCEP, and Hydrant
	invalidNameTestCases := []struct {
		testName   string
		name       string
		errMessage string
	}{
		{
			testName:   "empty",
			name:       "",
			errMessage: "name cannot be empty",
		},
		{
			testName:   "NDES",
			name:       "NDES",
			errMessage: "CA name cannot be NDES",
		},
		{
			testName:   "too long",
			name:       strings.Repeat("a", 256),
			errMessage: "CA name cannot be longer than ",
		},
		{
			testName:   "invalid characters",
			name:       "a/b",
			errMessage: "Only letters, numbers and underscores allowed",
		},
	}

	// common invalid URL test cases for DigiCert, Custom SCEP, and Hydrant
	invalidURLTestCases := []struct {
		testName   string
		url        string
		errMessage string
	}{
		{
			testName:   "empty",
			url:        "",
			errMessage: "Invalid URL",
		},
		{
			testName:   "non-http",
			url:        "nonhttp://bad.com",
			errMessage: "URL scheme must be https or http",
		},
	}

	t.Run("ndes", func(t *testing.T) {
		checkNDESApplied := func(t *testing.T, expectNDES *fleet.NDESSCEPProxyCA) {
			cas, err := s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
			require.NoError(t, err)

			if expectNDES != nil {
				require.NotNil(t, cas.NDESSCEP)
				require.NotZero(t, cas.NDESSCEP.ID)
				require.Equal(t, expectNDES.URL, cas.NDESSCEP.URL)
				require.Equal(t, expectNDES.AdminURL, cas.NDESSCEP.AdminURL)
				require.Equal(t, expectNDES.Username, cas.NDESSCEP.Username)
				require.Equal(t, expectNDES.Password, cas.NDESSCEP.Password)

			} else {
				require.Nil(t, cas.NDESSCEP)
			}
		}

		t.Run("invalid SCEP URL", func(t *testing.T) {
			testCopy := goodNDESSCEPCA
			testCopy.URL = "://invalid-url"

			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)

			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.ndes_scep_proxy")
			require.Contains(t, errMsg, "Invalid NDES SCEP URL")
			checkNDESApplied(t, nil)
		})

		t.Run("wrong SCEP URL", func(t *testing.T) {
			testCopy := goodNDESSCEPCA
			testCopy.URL = "https://new2.com/mscep/mscep.dll"

			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)

			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusBadRequest)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.ndes_scep_proxy")
			require.Contains(t, errMsg, "couldn't find the CA at the specified URL")
			checkNDESApplied(t, nil)
		})

		t.Run("empty password", func(t *testing.T) {
			testCopy := goodNDESSCEPCA
			testCopy.Password = ""

			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)

			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.ndes_scep_proxy")
			require.Contains(t, errMsg, "NDES SCEP password cannot be empty.")
			checkNDESApplied(t, nil)
		})

		t.Run("delete", func(t *testing.T) {
			t.Run("nil NDES deletes existing", func(t *testing.T) {
				// create a CA to delete
				req, err := newApplyRequest(goodNDESSCEPCA, false)
				require.NoError(t, err)
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
				checkNDESApplied(t, &goodNDESSCEPCA)

				// try dry run of deletion by making an apply request where NDES is nil and dry run is true
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{
					CertificateAuthorities: fleet.GroupedCertificateAuthorities{
						NDESSCEP: nil,
					},
					DryRun: true,
				}, http.StatusOK)
				checkNDESApplied(t, &goodNDESSCEPCA) // prior ndes should still exist

				// now delete it by making an apply request where NDES is nil and dry run is false
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{
					CertificateAuthorities: fleet.GroupedCertificateAuthorities{
						NDESSCEP: nil,
					},
					DryRun: false,
				}, http.StatusOK)
				checkNDESApplied(t, nil) // prior ndes deleted
			})

			t.Run("empty NDES deletes existing", func(t *testing.T) {
				// create a CA to delete
				req, err := newApplyRequest(goodNDESSCEPCA, false)
				require.NoError(t, err)
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
				checkNDESApplied(t, &goodNDESSCEPCA)

				// try dry run of deletion by making an apply request where NDES is an empty struct and dry run is true
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{
					CertificateAuthorities: fleet.GroupedCertificateAuthorities{
						NDESSCEP: &fleet.NDESSCEPProxyCA{},
					},
					DryRun: true,
				}, http.StatusOK)
				checkNDESApplied(t, &goodNDESSCEPCA) // prior ndes should still exist

				// now delete it by making an apply request where NDES is an empty struct
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{
					CertificateAuthorities: fleet.GroupedCertificateAuthorities{
						NDESSCEP: &fleet.NDESSCEPProxyCA{},
					},
					DryRun: false,
				}, http.StatusOK)
				checkNDESApplied(t, nil) // prior ndes deleted
			})

			t.Run("json null NDES deletes existing", func(t *testing.T) {
				// create a CA to delete
				req, err := newApplyRequest(goodNDESSCEPCA, false)
				require.NoError(t, err)
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
				checkNDESApplied(t, &goodNDESSCEPCA)

				// mock apply request where API user sends json null
				r := map[string]interface{}{
					"certificate_authorities": map[string]interface{}{
						"ndes_scep_proxy": nil,
					},
					"dry_run": true,
				}

				// first try a dry run
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", r, http.StatusOK)
				checkNDESApplied(t, &goodNDESSCEPCA) // prior ndes should still exist

				// now make a non-dry run apply request where NDES is nil, which should delete the
				// existing CA
				r["dry_run"] = false
				_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", r, http.StatusOK)
				checkNDESApplied(t, nil) // prior ndes should be deleted
			})
		})

		t.Run("happy path add then update then delete", func(t *testing.T) {
			checkNDESApplied(t, nil)

			req, err := newApplyRequest(goodNDESSCEPCA, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			checkNDESApplied(t, nil) // dry run
			req.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			checkNDESApplied(t, &goodNDESSCEPCA)

			// update
			testCopy := goodNDESSCEPCA
			testCopy.Password = "new-password"
			req, err = newApplyRequest(testCopy, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			checkNDESApplied(t, &goodNDESSCEPCA) // dry run should not change anything
			req.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			checkNDESApplied(t, &testCopy)

			// delete
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: true}, http.StatusOK)
			checkNDESApplied(t, &testCopy) // dry run should not change anything
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: false}, http.StatusOK)
			checkNDESApplied(t, nil) // prior ndes should be deleted
		})
	})

	t.Run("digicert", func(t *testing.T) {
		t.Run("invalid name", func(t *testing.T) {
			// run common invalid name test cases
			for _, tc := range invalidNameTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodDigiCertCA
					testCopy.Name = tc.name
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.digicert")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})

		// run common invalid URL test cases
		t.Run("invalid url", func(t *testing.T) {
			for _, tc := range invalidURLTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodDigiCertCA
					testCopy.URL = tc.url
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.digicert")
					if tc.errMessage == "Invalid URL" {
						require.Contains(t, errMsg, "Invalid DigiCert URL")
					} else {
						require.Contains(t, errMsg, tc.errMessage)
					}
				})
			}
		})

		// run additional duplicate name scenarios
		t.Run("duplicate names", func(t *testing.T) {
			// create one of each CA
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// try to create digicert with same name as another digicert
			testCopy := goodDigiCertCA
			testCopy.CertificateSeatID = "some-other-seat-id"
			duplicateReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA, testCopy},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
				},
				DryRun: false,
			}
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "name is already used by another DigiCert certificate authority")

			// try to create digicert with same name as another custom scep
			testCopy = goodDigiCertCA
			testCopy.Name = goodCustomSCEPCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA, testCopy},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create digicert with same name as another hydrant
			testCopy = goodDigiCertCA
			testCopy.Name = goodHydrantCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA, testCopy},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create digicert with same name as another smallstep
			testCopy = goodDigiCertCA
			testCopy.Name = goodSmallstepCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA, testCopy},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)
		})

		t.Run("digicert more than 1 user principal name", func(t *testing.T) {
			testCopy := goodDigiCertCA
			testCopy.CertificateUserPrincipalNames = []string{"user1", "user2"}
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)

			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "only one item can be added to certificate_user_principal_names")
		})

		t.Run("digicert empty user principal name", func(t *testing.T) {
			testCopy := goodDigiCertCA
			testCopy.CertificateUserPrincipalNames = []string{" "}
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)

			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "certificate_user_principal_name cannot be empty if specified")
		})

		t.Run("digicert Fleet vars in user principal name", func(t *testing.T) {
			// allowed usage
			testCopy := goodDigiCertCA
			testCopy.CertificateUserPrincipalNames = []string{"$FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + " ${FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "}"}
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// disallowed usage
			testCopy.CertificateUserPrincipalNames = []string{"$FLEET_VAR_BOZO"}
			req, err = newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "FLEET_VAR_BOZO is not allowed")
		})

		t.Run("digicert Fleet vars in common name", func(t *testing.T) {
			// allowed usage
			testCopy := goodDigiCertCA
			testCopy.CertificateCommonName = "${FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + "}${FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "}"
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// disallowed usage
			testCopy.CertificateCommonName = "$FLEET_VAR_BOZO"
			req, err = newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "FLEET_VAR_BOZO is not allowed")
		})

		t.Run("digicert Fleet vars in seat id", func(t *testing.T) {
			// allowed usage
			testCopy := goodDigiCertCA
			testCopy.CertificateSeatID = "${FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + "}${FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "}"
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// disallowed usage
			testCopy.CertificateSeatID = "$FLEET_VAR_BOZO"
			req, err = newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "FLEET_VAR_BOZO is not allowed")
		})

		t.Run("digicert API token not set", func(t *testing.T) {
			testCopy := goodDigiCertCA
			testCopy.APIToken = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "Invalid API token. Please correct and try again.")

			// try again with masked password, same as if it was not set
			testCopy.APIToken = fleet.MaskedPassword
			req, err = newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "Invalid API token. Please correct and try again.")
		})

		t.Run("digicert common name not set", func(t *testing.T) {
			testCopy := goodDigiCertCA
			testCopy.CertificateCommonName = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "Common Name (CN) cannot be empty")
		})

		t.Run("digicert seat id not set", func(t *testing.T) {
			testCopy := goodDigiCertCA
			testCopy.CertificateSeatID = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.digicert")
			require.Contains(t, errMsg, "Seat ID cannot be empty")
		})

		t.Run("digicert happy path with activities add then modify then delete", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			req1, err := newApplyRequest(goodDigiCertCA, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // dry run should not change anything
			req1.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // now it should be applied
			wantAdded := fleet.ActivityAddedDigiCert{
				Name: goodDigiCertCA.Name,
			}
			id := s.lastActivityMatches(wantAdded.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantAdded.Name), 0)

			testCopy := goodDigiCertCA
			testCopy.CertificateCommonName = "new-common-name"
			req2, err := newApplyRequest(testCopy, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // dry run should not change anything
			s.lastActivityMatches(wantAdded.ActivityName(), "", id) // no new activity yet
			req2.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // now it should be applied
			wantEdited := fleet.ActivityEditedDigiCert{
				Name: goodDigiCertCA.Name,
			}
			s.lastActivityMatches(wantEdited.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantEdited.Name), 0)
			s.lastActivityOfTypeMatches(wantAdded.ActivityName(), "", id) // last "added" activity is the prior one

			// sending empty CAs deletes existing one
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: true}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // dry run should not change anything
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: false}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // now delete should be applied
			wantDeleted := fleet.ActivityDeletedDigiCert{
				Name: goodDigiCertCA.Name,
			}
			s.lastActivityMatches(wantDeleted.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantDeleted.Name), 0)
		})

		t.Run("digicert happy path add one, delete one, modify one", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			// setup the test by creating two CAs
			test1 := goodDigiCertCA
			test2 := goodDigiCertCA
			test2.Name = "VALID_DIGICERT_CA_2"
			initialReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert: []fleet.DigiCertCA{test1, test2},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", initialReq, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, initialReq.CertificateAuthorities)

			// add third
			test3 := goodDigiCertCA
			test3.Name = "VALID_DIGICERT_CA_3"
			// modify first
			test1.CertificateCommonName = "new-common-name"

			// new request will modify test1, add test3, and delete test2
			modifiedReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert: []fleet.DigiCertCA{test1, test3},
				},
				DryRun: true,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", modifiedReq, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, initialReq.CertificateAuthorities) // dry run should not change anything
			modifiedReq.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", modifiedReq, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, modifiedReq.CertificateAuthorities) // now it should be applied

			// delete the rest
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", fleet.GroupedCertificateAuthorities{}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})
		})
	})

	t.Run("custom_scep_proxy", func(t *testing.T) {
		// run common invalid name test cases
		t.Run("invalid name", func(t *testing.T) {
			for _, tc := range invalidNameTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodCustomSCEPCA
					testCopy.Name = tc.name
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.custom_scep_proxy")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})
		// run common invalid url test cases
		t.Run("invalid url", func(t *testing.T) {
			for _, tc := range invalidURLTestCases {
				testCopy := goodCustomSCEPCA
				testCopy.URL = tc.url

				req, err := newApplyRequest(testCopy, false)
				require.NoError(t, err)
				res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
				errMsg := extractServerErrorText(res.Body)
				require.Contains(t, errMsg, "certificate_authorities.custom_scep_proxy")
				if tc.errMessage == "Invalid URL" {
					require.Contains(t, errMsg, "Invalid SCEP URL")
				} else {
					require.Contains(t, errMsg, tc.errMessage)
				}
			}
		})

		// run additional duplicate name scenarios
		t.Run("duplicate names", func(t *testing.T) {
			// create one of each CA
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// try to create custom scep with same name as another custom scep
			testCopy := goodCustomSCEPCA
			testCopy.URL = "https://example.com"
			duplicateReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA, testCopy},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.custom_scep_proxy")
			require.Contains(t, errMsg, "name is already used by another Custom SCEP Proxy certificate authority")

			// try to create custom scep with same name as the digicert. Should not error
			testCopy = goodCustomSCEPCA
			testCopy.Name = goodDigiCertCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA, testCopy},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create custom scep with same name as the Hydrant CA. Should not error
			testCopy = goodCustomSCEPCA
			testCopy.Name = goodHydrantCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA, testCopy},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create custom scep with same name as the Smallstep CA. Should not error
			testCopy = goodCustomSCEPCA
			testCopy.Name = goodSmallstepCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA, testCopy},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)
		})

		t.Run("custom_scep challenge not set", func(t *testing.T) {
			testCopy := goodCustomSCEPCA
			testCopy.Challenge = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.custom_scep_proxy")
			require.Contains(t, errMsg, "Custom SCEP Proxy challenge cannot be empty")

			// try with masked password, same as if it was not set
			testCopy.Challenge = fleet.MaskedPassword
			req, err = newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.custom_scep_proxy")
			require.Contains(t, errMsg, "Custom SCEP Proxy challenge cannot be empty")
		})

		t.Run("custom scep happy path with activities add then modify then delete", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			req1, err := newApplyRequest(goodCustomSCEPCA, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // dry run
			req1.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities)
			wantAdded := fleet.ActivityAddedCustomSCEPProxy{
				Name: goodCustomSCEPCA.Name,
			}
			id := s.lastActivityMatches(wantAdded.ActivityName(), fmt.Sprintf(`{"name":"%s"}`, wantAdded.Name), 0)

			testCopy := goodCustomSCEPCA
			testCopy.Challenge = "some-new-challenge"
			req2, err := newApplyRequest(testCopy, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // dry run so no changes
			s.lastActivityMatches(wantAdded.ActivityName(), "", id) // no new activity
			req2.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // changes now applied
			wantEdited := fleet.ActivityEditedCustomSCEPProxy{
				Name: goodCustomSCEPCA.Name,
			}
			s.lastActivityMatches(wantEdited.ActivityName(), fmt.Sprintf(`{"name":"%s"}`, wantEdited.Name), 0) // last "edited" activity is the prior one
			s.lastActivityOfTypeMatches(wantAdded.ActivityName(), "", id)                                      // last "added" activity is the prior one

			// sending empty CAs deletes existing one
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: true}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // dry run should not change anything
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: false}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})
			wantDeleted := fleet.ActivityDeletedCustomSCEPProxy{
				Name: goodCustomSCEPCA.Name,
			}
			s.lastActivityMatches(wantDeleted.ActivityName(), fmt.Sprintf(`{"name":"%s"}`, wantDeleted.Name), 0)
		})

		t.Run("custom scep happy path add one, delete one, modify one", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			// setup the test by creating two CAs
			test1 := goodCustomSCEPCA
			test2 := goodCustomSCEPCA
			test2.Name = "VALID_CUSTOM_SCEP_CA_2"
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					CustomScepProxy: []fleet.CustomSCEPProxyCA{test1, test2},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			// add third
			test3 := goodCustomSCEPCA
			test3.Name = "VALID_CUSTOM_SCEP_CA_3"
			// modify first
			test1.Challenge = "new-challenge"

			// new request will modify test1, add test3, and delete test2
			req2 := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					CustomScepProxy: []fleet.CustomSCEPProxyCA{test1, test3},
				},
				DryRun: true,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities) // dry run should not change anything
			req2.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities)

			// delete the rest
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", fleet.GroupedCertificateAuthorities{}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})
		})
	})

	t.Run("smallstep", func(t *testing.T) {
		// run common invalid name test cases
		t.Run("invalid name", func(t *testing.T) {
			for _, tc := range invalidNameTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodSmallstepCA
					testCopy.Name = tc.name
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.smallstep")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})
		// run common invalid url test cases
		t.Run("invalid url", func(t *testing.T) {
			for _, tc := range invalidURLTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodSmallstepCA
					testCopy.URL = tc.url
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.smallstep")
					if tc.errMessage == "Invalid URL" {
						require.Contains(t, errMsg, "Invalid Smallstep SCEP URL")
					} else {
						require.Contains(t, errMsg, tc.errMessage)
					}
				})
			}
		})

		t.Run("duplicate names", func(t *testing.T) {
			// create one of each CA
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// try to create smallstep with same name as another smallstep
			testCopy := goodSmallstepCA
			testCopy.URL = "https://example.com"
			duplicateReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA, testCopy},
				},
				DryRun: false,
			}
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.smallstep")
			require.Contains(t, errMsg, "name is already used by another Smallstep certificate authority")

			// try to create smallstep with same name as another digicert
			testCopy = goodSmallstepCA
			testCopy.Name = goodDigiCertCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA, testCopy},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create smallstep with same name as another hydrant
			testCopy = goodSmallstepCA
			testCopy.Name = goodHydrantCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA, testCopy},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create smallstep with same name as another custom scep
			testCopy = goodSmallstepCA
			testCopy.Name = goodCustomSCEPCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA, testCopy},
				},
				DryRun: false,
			}

			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)
		})

		t.Run("smallstep url not set", func(t *testing.T) {
			testCopy := goodSmallstepCA
			testCopy.URL = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.smallstep")
			require.Contains(t, errMsg, "Invalid Smallstep SCEP URL")
		})

		t.Run("smallstep challenge url not set", func(t *testing.T) {
			testCopy := goodSmallstepCA
			testCopy.ChallengeURL = ""

			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusBadRequest)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.smallstep")
			require.Contains(t, errMsg, "Invalid challenge URL or credentials")
		})

		t.Run("smallstep username not set", func(t *testing.T) {
			testCopy := goodSmallstepCA
			testCopy.Username = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.smallstep")
			require.Contains(t, errMsg, "Smallstep username cannot be empty")
		})

		t.Run("smallstep password not set", func(t *testing.T) {
			testCopy := goodSmallstepCA
			testCopy.Password = ""
			req, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.smallstep")
			require.Contains(t, errMsg, "Smallstep password cannot be empty")

			// try with masked password, same as if it was not set
			testCopy.Password = fleet.MaskedPassword
			req, err = newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.smallstep")
			require.Contains(t, errMsg, "Smallstep password cannot be empty")
		})

		t.Run("smallstep happy path with activities add then modify then delete", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			req1, err := newApplyRequest(goodSmallstepCA, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // dry run should not change anything
			req1.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // now it should be applied
			wantAdded := fleet.ActivityAddedSmallstep{
				Name: goodSmallstepCA.Name,
			}
			id := s.lastActivityMatches(wantAdded.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantAdded.Name), 0)

			testCopy := goodSmallstepCA
			testCopy.Username = "username"
			req2, err := newApplyRequest(testCopy, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // dry run should not change anything
			s.lastActivityMatches(wantAdded.ActivityName(), "", id) // no new activity yet
			req2.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // now it should be applied
			wantEdited := fleet.ActivityEditedSmallstep{
				Name: goodSmallstepCA.Name,
			}
			s.lastActivityMatches(wantEdited.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantEdited.Name), 0)
			s.lastActivityOfTypeMatches(wantAdded.ActivityName(), "", id) // last "added" activity is the prior one

			// sending empty CAs deletes existing one
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: true}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // dry run should not change anything
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: false}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // now delete should be applied
			wantDeleted := fleet.ActivityDeletedSmallstep{
				Name: goodSmallstepCA.Name,
			}
			s.lastActivityMatches(wantDeleted.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantDeleted.Name), 0)
		})

		t.Run("smallstep happy path add one, delete one, modify one", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			// setup the test by creating two CAs
			test1 := goodSmallstepCA
			test2 := goodSmallstepCA
			test2.Name = "VALID_SMALLSTEP_CA_2"
			initialReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					Smallstep: []fleet.SmallstepSCEPProxyCA{test1, test2},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", initialReq, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, initialReq.CertificateAuthorities)

			// add third
			test3 := goodSmallstepCA
			test3.Name = "VALID_SMALLSTEP_CA_3"
			// modify first
			test1.Username = "username"

			// new request will modify test1, add test3, and delete test2
			modifiedReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					Smallstep: []fleet.SmallstepSCEPProxyCA{test1, test3},
				},
				DryRun: true,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", modifiedReq, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, initialReq.CertificateAuthorities) // dry run should not change anything
			modifiedReq.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", modifiedReq, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, modifiedReq.CertificateAuthorities) // now it should be applied

			// delete the rest
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", fleet.GroupedCertificateAuthorities{}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})
		})
	})

	t.Run("hydrant", func(t *testing.T) {
		// run common invalid name test cases
		t.Run("invalid name", func(t *testing.T) {
			for _, tc := range invalidNameTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodHydrantCA
					testCopy.Name = tc.name
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.hydrant")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})
		// run common invalid url test cases
		t.Run("invalid url", func(t *testing.T) {
			for _, tc := range invalidURLTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodHydrantCA
					testCopy.URL = tc.url
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.hydrant")
					if tc.errMessage == "Invalid URL" {
						require.Contains(t, errMsg, "Invalid Hydrant URL")
					} else {
						require.Contains(t, errMsg, tc.errMessage)
					}
				})
			}
		})

		// run additional duplicate name scenarios
		t.Run("duplicate names", func(t *testing.T) {
			// create one of each CA
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// try to create hydrant with same name as another hydrant
			testCopy := goodHydrantCA
			testCopy.ClientID = "some-other-client-id"
			duplicateReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA, testCopy},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.hydrant")
			require.Contains(t, errMsg, "name is already used by another Hydrant certificate authority")

			// try to create hydrant with same name as another digicert
			testCopy = goodHydrantCA
			testCopy.Name = goodDigiCertCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA, testCopy},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create hydrant with same name as another custom scep
			testCopy = goodHydrantCA
			testCopy.Name = goodCustomSCEPCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA, testCopy},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create hydrant with same name as another smallstep
			testCopy = goodHydrantCA
			testCopy.Name = goodSmallstepCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA, testCopy},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)
		})

		// TODO(hca): hydrant happy path and other specific tests
	})

	t.Run("google cas", func(t *testing.T) {
		t.Run("invalid name", func(t *testing.T) {
			for _, tc := range invalidNameTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodGoogleCASCA
					testCopy.Name = tc.name
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.google_cas")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})

		t.Run("invalid fields", func(t *testing.T) {
			for _, tc := range []struct {
				testName   string
				mutate     func(ca *fleet.GoogleCASCA)
				errMessage string
			}{
				// The full set of field checks is in the service unit tests; these show they run on the GitOps path.
				{"pool with a slash", func(ca *fleet.GoogleCASCA) { ca.CAPool = "pool/certificates/other" }, "The CA pool can only contain"},
				{"key is not json", func(ca *fleet.GoogleCASCA) { ca.APIKeyJSON = "nope" }, "The service account key must be a JSON object."},
			} {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodGoogleCASCA
					tc.mutate(&testCopy)
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.google_cas")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})

		t.Run("duplicate names", func(t *testing.T) {
			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// two Google CAS CAs with the same name are rejected
			testCopy := goodGoogleCASCA
			testCopy.CAPool = "some-other-pool"
			duplicateReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{GoogleCAS: []fleet.GoogleCASCA{goodGoogleCASCA, testCopy}},
			}
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.google_cas")
			require.Contains(t, errMsg, "name is already used by another Google CAS certificate authority")

			// the same name as a CA of another type is fine
			testCopy = goodGoogleCASCA
			testCopy.Name = goodHydrantCA.Name
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					Hydrant:   []fleet.HydrantCA{goodHydrantCA},
					GoogleCAS: []fleet.GoogleCASCA{goodGoogleCASCA, testCopy},
				},
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)
		})

		t.Run("google cas happy path with activities add then modify then delete", func(t *testing.T) {
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{})

			var verified []fleet.GoogleCASCA
			s.googleCASService.VerifyCAPoolFunc = func(_ context.Context, ca fleet.GoogleCASCA) error {
				verified = append(verified, ca)
				return nil
			}
			t.Cleanup(func() {
				s.googleCASService.VerifyCAPoolFunc = func(_ context.Context, _ fleet.GoogleCASCA) error { return nil }
			})

			req1, err := newApplyRequest(goodGoogleCASCA, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // dry run should not change anything
			req1.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req1, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // now it should be applied
			require.Len(t, verified, 2)                             // the dry run verifies the pool too
			require.Equal(t, goodGoogleCASCA, verified[0])
			wantAdded := fleet.ActivityAddedGoogleCAS{Name: goodGoogleCASCA.Name}
			id := s.lastActivityMatches(wantAdded.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantAdded.Name), 0)

			testCopy := goodGoogleCASCA
			testCopy.CAPool = "other-pool"
			req2, err := newApplyRequest(testCopy, true)
			require.NoError(t, err)
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req1.CertificateAuthorities) // dry run should not change anything
			s.lastActivityMatches(wantAdded.ActivityName(), "", id) // no new activity yet
			req2.DryRun = false
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // now it should be applied
			wantEdited := fleet.ActivityEditedGoogleCAS{Name: goodGoogleCASCA.Name}
			editedID := s.lastActivityMatches(wantEdited.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantEdited.Name), 0)
			s.lastActivityOfTypeMatches(wantAdded.ActivityName(), "", id) // last "added" activity is the prior one

			// applying the same config again changes nothing, asks Google nothing and records no activity
			verified = nil
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req2, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities)
			require.Empty(t, verified)
			s.lastActivityMatches(wantEdited.ActivityName(), "", editedID)

			// a CA pool Google rejects leaves the stored CA alone
			s.googleCASService.VerifyCAPoolFunc = func(_ context.Context, _ fleet.GoogleCASCA) error { return errors.New("no access") }
			testCopy.CAPool = "rejected-pool"
			req3, err := newApplyRequest(testCopy, false)
			require.NoError(t, err)
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req3, http.StatusBadRequest)
			require.Contains(t, extractServerErrorText(res.Body), "certificate_authorities.google_cas: no access")
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities)
			s.googleCASService.VerifyCAPoolFunc = func(_ context.Context, _ fleet.GoogleCASCA) error { return nil }

			// sending empty CAs deletes existing one
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: true}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req2.CertificateAuthorities) // dry run should not change anything
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{DryRun: false}, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, fleet.GroupedCertificateAuthorities{}) // now delete should be applied
			wantDeleted := fleet.ActivityDeletedGoogleCAS{Name: goodGoogleCASCA.Name}
			s.lastActivityMatches(wantDeleted.ActivityName(), fmt.Sprintf(`{"name":%q}`, wantDeleted.Name), 0)
		})
	})

	t.Run("custom est", func(t *testing.T) {
		// run common invalid name test cases
		t.Run("invalid name", func(t *testing.T) {
			for _, tc := range invalidNameTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodESTCA
					testCopy.Name = tc.name
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.custom_est_proxy")
					require.Contains(t, errMsg, tc.errMessage)
				})
			}
		})
		// run common invalid url test cases
		t.Run("invalid url", func(t *testing.T) {
			for _, tc := range invalidURLTestCases {
				t.Run(tc.testName, func(t *testing.T) {
					testCopy := goodESTCA
					testCopy.URL = tc.url
					req, err := newApplyRequest(testCopy, false)
					require.NoError(t, err)
					res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusUnprocessableEntity)
					errMsg := extractServerErrorText(res.Body)
					require.Contains(t, errMsg, "certificate_authorities.custom_est_proxy")
					if tc.errMessage == "Invalid URL" {
						require.Contains(t, errMsg, "Invalid EST URL")
					} else {
						require.Contains(t, errMsg, tc.errMessage)
					}
				})
			}
		})

		// run additional duplicate name scenarios
		t.Run("duplicate names", func(t *testing.T) {
			// create one of each CA
			req := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
					return nil
				})
			})

			// try to create est ca proxy with same name as another est ca
			testCopy := goodESTCA
			testCopy.Username = "some-other-client-id"
			duplicateReq := batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA, testCopy},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "certificate_authorities.custom_est_proxy")
			require.Contains(t, errMsg, "name is already used by another Custom EST Proxy certificate authority")

			// try to create a custom est ca with same name as another digicert
			testCopy = goodESTCA
			testCopy.Name = goodDigiCertCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA, testCopy},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create est with same name as another custom scep
			testCopy = goodESTCA
			testCopy.Name = goodCustomSCEPCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA, testCopy},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)

			// try to create eset ca with same name as another smallstep
			testCopy = goodESTCA
			testCopy.Name = goodSmallstepCA.Name
			duplicateReq = batchApplyCertificateAuthoritiesRequest{
				CertificateAuthorities: fleet.GroupedCertificateAuthorities{
					DigiCert:        []fleet.DigiCertCA{goodDigiCertCA},
					CustomScepProxy: []fleet.CustomSCEPProxyCA{goodCustomSCEPCA},
					Hydrant:         []fleet.HydrantCA{goodHydrantCA},
					EST:             []fleet.ESTProxyCA{goodESTCA, testCopy},
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)
		})
	})

	t.Run("creating a Hydrant CA through the REST API records its name", func(t *testing.T) {
		hydrant := goodHydrantCA
		hydrant.Name = "REST_HYDRANT"
		var resp createCertificateAuthorityResponse
		s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", fleet.CertificateAuthorityPayload{Hydrant: &hydrant}, http.StatusOK, &resp)
		s.lastActivityMatches(fleet.ActivityAddedHydrant{}.ActivityName(), `{"name":"REST_HYDRANT"}`, 0)
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", resp.ID), nil, http.StatusNoContent)
	})

	t.Run("names that differ only in case are the same CA", func(t *testing.T) {
		t.Cleanup(func() {
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, _ = q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
				return nil
			})
		})
		named := func(names ...string) fleet.GroupedCertificateAuthorities {
			var cas fleet.GroupedCertificateAuthorities
			for _, name := range names {
				digicert, customSCEP, hydrant, est, smallstep, googleCAS := goodDigiCertCA, goodCustomSCEPCA, goodHydrantCA, goodESTCA, goodSmallstepCA, goodGoogleCASCA
				digicert.Name, customSCEP.Name, hydrant.Name, est.Name, smallstep.Name, googleCAS.Name = name, name, name, name, name, name
				cas.DigiCert = append(cas.DigiCert, digicert)
				cas.CustomScepProxy = append(cas.CustomScepProxy, customSCEP)
				cas.Hydrant = append(cas.Hydrant, hydrant)
				cas.EST = append(cas.EST, est)
				cas.Smallstep = append(cas.Smallstep, smallstep)
				cas.GoogleCAS = append(cas.GoogleCAS, googleCAS)
			}
			return cas
		}
		apply := func(cas fleet.GroupedCertificateAuthorities, status int) *http.Response {
			return s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{CertificateAuthorities: cas}, status)
		}

		apply(named("CASE_X"), http.StatusOK)
		lastID := s.lastActivityMatches("", "", 0)

		// Deletes are enabled, as the request doesn't set skip_deletes: each CA is renamed, not deleted.
		apply(named("case_x"), http.StatusOK)
		s.checkAppliedCAs(t, s.ds, named("case_x"))
		var activities listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities, "order_key", "id", "order_direction", "desc", "per_page", "20")
		var newTypes []string
		for _, a := range activities.Activities {
			if a.ID > lastID {
				newTypes = append(newTypes, a.Type)
			}
		}
		require.ElementsMatch(t, []string{
			fleet.ActivityEditedDigiCert{}.ActivityName(), fleet.ActivityEditedCustomSCEPProxy{}.ActivityName(),
			fleet.ActivityEditedHydrant{}.ActivityName(), fleet.ActivityEditedCustomESTProxy{}.ActivityName(),
			fleet.ActivityEditedSmallstep{}.ActivityName(), fleet.ActivityEditedGoogleCAS{}.ActivityName(),
		}, newTypes)

		duplicates := named("DUP_X", "dup_x")
		for _, tc := range []struct {
			caType, displayType string
			cas                 fleet.GroupedCertificateAuthorities
		}{
			{"digicert", "DigiCert", fleet.GroupedCertificateAuthorities{DigiCert: duplicates.DigiCert}},
			{"custom_scep_proxy", "Custom SCEP Proxy", fleet.GroupedCertificateAuthorities{CustomScepProxy: duplicates.CustomScepProxy}},
			{"hydrant", "Hydrant", fleet.GroupedCertificateAuthorities{Hydrant: duplicates.Hydrant}},
			{"custom_est_proxy", "Custom EST Proxy", fleet.GroupedCertificateAuthorities{EST: duplicates.EST}},
			{"smallstep", "Smallstep", fleet.GroupedCertificateAuthorities{Smallstep: duplicates.Smallstep}},
			{"google_cas", "Google CAS", fleet.GroupedCertificateAuthorities{GoogleCAS: duplicates.GoogleCAS}},
		} {
			errMsg := extractServerErrorText(apply(tc.cas, http.StatusUnprocessableEntity).Body)
			require.Contains(t, errMsg, "certificate_authorities."+tc.caType)
			require.Contains(t, errMsg, "name is already used by another "+tc.displayType+" certificate authority")
		}
		s.checkAppliedCAs(t, s.ds, named("case_x"))
	})
}

func (s *integrationMDMTestSuite) checkAppliedCAs(t *testing.T, ds fleet.Datastore, expectedCAs fleet.GroupedCertificateAuthorities) {
	var gotResp getCertificateAuthoritiesSpecResponse
	s.DoJSON("GET", "/api/v1/fleet/spec/certificate_authorities?include_secrets=true", nil, http.StatusOK, &gotResp)
	gotCAs := gotResp.CertificateAuthorities

	if len(expectedCAs.DigiCert) != 0 {
		assert.Len(t, gotCAs.DigiCert, len(expectedCAs.DigiCert))
		wantByName := make(map[string]fleet.DigiCertCA)
		gotByName := make(map[string]fleet.DigiCertCA)
		for _, ca := range expectedCAs.DigiCert {
			wantByName[ca.Name] = ca
		}
		for _, ca := range gotCAs.DigiCert {
			ca.ID = 0 // ignore IDs when comparing
			gotByName[ca.Name] = ca
		}
		assert.Equal(t, wantByName, gotByName)
	} else {
		assert.Empty(t, gotCAs.DigiCert)
	}

	if len(expectedCAs.CustomScepProxy) != 0 {
		assert.Len(t, gotCAs.CustomScepProxy, len(expectedCAs.CustomScepProxy))
		wantByName := make(map[string]fleet.CustomSCEPProxyCA)
		gotByName := make(map[string]fleet.CustomSCEPProxyCA)
		for _, ca := range expectedCAs.CustomScepProxy {
			wantByName[ca.Name] = ca
		}
		for _, ca := range gotCAs.CustomScepProxy {
			ca.ID = 0 // ignore IDs when comparing
			gotByName[ca.Name] = ca
		}
		assert.Equal(t, wantByName, gotByName)
	} else {
		assert.Empty(t, gotCAs.CustomScepProxy)
	}

	if len(expectedCAs.Hydrant) != 0 {
		assert.Len(t, gotCAs.Hydrant, len(expectedCAs.Hydrant))
		wantByName := make(map[string]fleet.HydrantCA)
		gotByName := make(map[string]fleet.HydrantCA)
		for _, ca := range expectedCAs.Hydrant {
			wantByName[ca.Name] = ca
		}
		for _, ca := range gotCAs.Hydrant {
			ca.ID = 0 // ignore IDs when comparing
			gotByName[ca.Name] = ca
		}
		assert.Equal(t, wantByName, gotByName)
	} else {
		assert.Empty(t, gotCAs.Hydrant)
	}

	if len(expectedCAs.EST) != 0 {
		assert.Len(t, gotCAs.EST, len(expectedCAs.EST))
		wantByName := make(map[string]fleet.ESTProxyCA)
		gotByName := make(map[string]fleet.ESTProxyCA)
		for _, ca := range expectedCAs.EST {
			wantByName[ca.Name] = ca
		}
		for _, ca := range gotCAs.EST {
			ca.ID = 0 // ignore IDs when comparing
			gotByName[ca.Name] = ca
		}
		assert.Equal(t, wantByName, gotByName)
	} else {
		assert.Empty(t, gotCAs.EST)
	}

	if len(expectedCAs.Smallstep) != 0 {
		assert.Len(t, gotCAs.Smallstep, len(expectedCAs.Smallstep))
		wantByName := make(map[string]fleet.SmallstepSCEPProxyCA)
		gotByName := make(map[string]fleet.SmallstepSCEPProxyCA)
		for _, ca := range expectedCAs.Smallstep {
			wantByName[ca.Name] = ca
		}
		for _, ca := range gotCAs.Smallstep {
			ca.ID = 0 // ignore IDs when comparing
			gotByName[ca.Name] = ca
		}
		assert.Equal(t, wantByName, gotByName)
	} else {
		assert.Empty(t, gotCAs.Smallstep)
	}

	if expectedCAs.NDESSCEP != nil {
		assert.NotNil(t, gotCAs.NDESSCEP)
		gotCAs.NDESSCEP.ID = 0 // ignore ID when comparing
		assert.Equal(t, expectedCAs.NDESSCEP, gotCAs.NDESSCEP)
	} else {
		assert.Empty(t, gotCAs.NDESSCEP)
	}

	if len(expectedCAs.GoogleCAS) != 0 {
		assert.Len(t, gotCAs.GoogleCAS, len(expectedCAs.GoogleCAS))
		wantByName := make(map[string]fleet.GoogleCASCA)
		gotByName := make(map[string]fleet.GoogleCASCA)
		for _, ca := range expectedCAs.GoogleCAS {
			wantByName[ca.Name] = ca
		}
		for _, ca := range gotCAs.GoogleCAS {
			ca.ID = 0 // ignore IDs when comparing
			gotByName[ca.Name] = ca
		}
		assert.Equal(t, wantByName, gotByName)
	} else {
		assert.Empty(t, gotCAs.GoogleCAS)
	}
}

func (s *integrationMDMTestSuite) TestGoogleCASCertificateAuthorities() {
	t := s.T()
	ctx := context.Background()
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM certificate_authorities")
		return err
	})
	t.Cleanup(func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "DELETE FROM certificate_authorities")
			return err
		})
	})

	var verified []fleet.GoogleCASCA
	var verifyErr error
	s.googleCASService.VerifyCAPoolFunc = func(_ context.Context, ca fleet.GoogleCASCA) error {
		verified = append(verified, ca)
		return verifyErr
	}

	const key = `{"type":"service_account","client_email":"sa@my-project.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----"}`
	createReq := func(name string) map[string]any {
		return map[string]any{"google_cas": map[string]any{
			// The key file's JSON as an object, the form GitOps produces; the edits below send it as a string.
			"name": name, "project": "my-project", "location": "us-central1", "ca_pool": "my-ca-pool", "api_key_json": json.RawMessage(key),
		}}
	}

	// Create.
	var createResp createCertificateAuthorityResponse
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", createReq("GOOGLE_WIFI"), http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	require.Equal(t, fleet.CATypeGoogleCAS, createResp.Type)
	s.lastActivityMatches(fleet.ActivityAddedGoogleCAS{}.ActivityName(), `{"name":"GOOGLE_WIFI"}`, 0)
	require.Len(t, verified, 1)
	require.Equal(t, "projects/my-project/locations/us-central1/caPools/my-ca-pool", verified[0].CAPoolResourceName())
	require.JSONEq(t, key, string(verified[0].APIKeyJSON))

	// The same name is already used.
	res := s.Do("POST", "/api/latest/fleet/certificate_authorities", createReq("GOOGLE_WIFI"), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `"GOOGLE_WIFI" name is already used by another Google CAS certificate authority`)

	// A CA pool Google rejects is not stored.
	verifyErr = errors.New("The service account can't access the CA pool.")
	res = s.Do("POST", "/api/latest/fleet/certificate_authorities", createReq("GOOGLE_REJECTED"), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "The service account can't access the CA pool.")
	verifyErr = nil

	// Only one CA type per request.
	twoTypes := createReq("GOOGLE_TWO")
	twoTypes["hydrant"] = map[string]any{"name": "H", "url": "https://example.com", "client_id": "a", "client_secret": "b"}
	s.Do("POST", "/api/latest/fleet/certificate_authorities", twoTypes, http.StatusBadRequest)

	// The list and the get never return the key.
	var listResp listCertificateAuthoritiesResponse
	listRes := s.DoRaw("GET", "/api/latest/fleet/certificate_authorities", nil, http.StatusOK)
	body, err := io.ReadAll(listRes.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &listResp))
	require.Len(t, listResp.CertificateAuthorities, 1)
	require.Equal(t, string(fleet.CATypeGoogleCAS), listResp.CertificateAuthorities[0].Type)
	require.NotContains(t, string(body), "PRIVATE KEY")

	var getResp getCertificateAuthorityResponse
	getRes := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", createResp.ID), nil, http.StatusOK)
	body, err = io.ReadAll(getRes.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &getResp))
	require.Equal(t, "my-project", *getResp.GoogleCASProject)
	require.Equal(t, "us-central1", *getResp.GoogleCASLocation)
	require.Equal(t, "my-ca-pool", *getResp.GoogleCASCAPool)
	require.Equal(t, fleet.MaskedPassword, *getResp.GoogleCASAPIKeyJSON)
	require.NotContains(t, string(body), "PRIVATE KEY")

	// Renaming doesn't need Google.
	verified = nil
	editURL := fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", createResp.ID)
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{"name": "GOOGLE_VPN"}}, http.StatusOK)
	s.lastActivityMatches(fleet.ActivityEditedGoogleCAS{}.ActivityName(), `{"name":"GOOGLE_VPN"}`, 0)
	require.Empty(t, verified)

	// Changing the pool, with the masked key, verifies with the stored key and keeps it.
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{"ca_pool": "other-pool", "api_key_json": fleet.MaskedPassword}}, http.StatusOK)
	require.Len(t, verified, 1)
	require.Equal(t, "other-pool", verified[0].CAPool)
	require.JSONEq(t, key, string(verified[0].APIKeyJSON))
	stored, err := s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, true)
	require.NoError(t, err)
	require.Equal(t, "other-pool", *stored.GoogleCASCAPool)
	require.JSONEq(t, key, *stored.GoogleCASAPIKeyJSON)

	// A replacement key Google rejects leaves the stored one in place.
	newKey := `{"type":"service_account","client_email":"other@my-project.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----"}`
	verifyErr = errors.New("The service account can't access the CA pool.")
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{"api_key_json": newKey}}, http.StatusBadRequest)
	verifyErr = nil
	stored, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, true)
	require.NoError(t, err)
	require.JSONEq(t, key, *stored.GoogleCASAPIKeyJSON)

	// A replacement key Google accepts replaces it.
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{"api_key_json": newKey}}, http.StatusOK)
	stored, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, true)
	require.NoError(t, err)
	require.Equal(t, newKey, *stored.GoogleCASAPIKeyJSON)
	editedID := s.lastActivityMatches(fleet.ActivityEditedGoogleCAS{}.ActivityName(), `{"name":"GOOGLE_VPN"}`, 0)

	// Sending nothing new, or the stored values again, asks Google nothing and records no activity.
	verified = nil
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{}}, http.StatusOK)
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{"name": "GOOGLE_VPN", "ca_pool": "other-pool", "api_key_json": fleet.MaskedPassword}}, http.StatusOK)
	require.Empty(t, verified)
	s.lastActivityMatches(fleet.ActivityEditedGoogleCAS{}.ActivityName(), "", editedID)

	// A malformed key is rejected before Google is asked.
	verified = nil
	s.Do("PATCH", editURL, map[string]any{"google_cas": map[string]any{"api_key_json": "nope"}}, http.StatusUnprocessableEntity)
	require.Empty(t, verified)

	// The type can't change.
	s.Do("PATCH", editURL, map[string]any{"hydrant": map[string]any{"name": "H"}}, http.StatusBadRequest)

	// Delete.
	s.Do("DELETE", editURL, nil, http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityDeletedGoogleCAS{}.ActivityName(), `{"name":"GOOGLE_VPN"}`, 0)
	_, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, false)
	require.Error(t, err)
}

// TestUpdateNDESCertificateAuthority covers PATCH /certificate_authorities/{id} for the NDES
// CA. The admin URL, username and password authenticate against NDES as a set, so a change to
// any of them has to be validated against the NDES server, not just a change to the admin URL.
func (s *integrationMDMTestSuite) TestUpdateNDESCertificateAuthority() {
	t := s.T()

	const (
		ndesUsername    = "ndes-username"
		ndesPassword    = "ndes-password"
		newNdesUsername = "new-ndes-username"
		newNdesPassword = "new-ndes-password" //nolint:gosec // G101: test value, not a real credential
	)

	scepServer := sceptest.NewTestSCEPServer(t)

	// an NDES admin server that only serves a challenge to the credentials above
	var authenticatedRequests atomic.Int64
	ndesAdminServer := sceptest.NewTestNDESAdminServerWithAuth(t, func(username, password string) bool {
		return (username == ndesUsername || username == newNdesUsername) &&
			(password == ndesPassword || password == newNdesPassword)
	}, &authenticatedRequests)

	var createResp createCertificateAuthorityResponse
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", fleet.CertificateAuthorityPayload{
		NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
			URL:      scepServer.URL + "/scep",
			AdminURL: ndesAdminServer.URL + "/mscep_admin/",
			Username: ndesUsername,
			Password: ndesPassword,
		},
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	caPath := fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", createResp.ID)

	patchNDES := func(update fleet.NDESSCEPProxyCAUpdatePayload, expectedStatus int) *http.Response {
		return s.Do("PATCH", caPath, fleet.CertificateAuthorityUpdatePayload{
			NDESSCEPProxyCAUpdatePayload: &update,
		}, expectedStatus)
	}

	requireStoredUsername := func(expected string) {
		var getResp getCertificateAuthorityResponse
		s.DoJSON("GET", caPath, getCertificateAuthorityRequest{}, http.StatusOK, &getResp)
		require.NotNil(t, getResp.Username)
		require.Equal(t, expected, *getResp.Username)
	}

	t.Run("username change with invalid credentials is rejected", func(t *testing.T) {
		// The password has to accompany a username change, so this is the smallest payload
		// the UI sends when only the username is edited.
		res := patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			Username: new("wrong-username"),
			Password: new(ndesPassword),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "Invalid NDES SCEP admin URL or credentials")

		requireStoredUsername(ndesUsername)
	})

	t.Run("password change with invalid credentials is rejected", func(t *testing.T) {
		res := patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			Password: new("wrong-password"),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "Invalid NDES SCEP admin URL or credentials")
	})

	t.Run("unreachable admin URL is reported as a connection problem", func(t *testing.T) {
		unreachableServer := httptest.NewServer(http.NotFoundHandler())
		unreachableServer.Close() // closed on purpose so the port refuses connections

		res := patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			AdminURL: new(unreachableServer.URL + "/mscep_admin/"),
			Password: new(ndesPassword),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "Couldn't connect to NDES SCEP admin URL")
	})

	t.Run("password change with valid credentials is validated and saved", func(t *testing.T) {
		before := authenticatedRequests.Load()

		patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			Password: new(newNdesPassword),
		}, http.StatusOK)

		// The update is only valid because Fleet asked the NDES server about it.
		require.Greater(t, authenticatedRequests.Load(), before)
	})

	t.Run("unchanged credentials are saved without asking the NDES server", func(t *testing.T) {
		before := authenticatedRequests.Load()

		patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			Username: new(ndesUsername),
			Password: new(newNdesPassword), // saved by the previous subtest
		}, http.StatusOK)

		require.Equal(t, before, authenticatedRequests.Load())
	})

	t.Run("username change with valid credentials is validated and saved", func(t *testing.T) {
		before := authenticatedRequests.Load()

		patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			Username: new(newNdesUsername),
			Password: new(newNdesPassword),
		}, http.StatusOK)

		require.Greater(t, authenticatedRequests.Load(), before)
		requireStoredUsername(newNdesUsername)
	})

	t.Run("masked password is rejected", func(t *testing.T) {
		before := authenticatedRequests.Load()

		// The GET endpoint returns the password masked. Changing NDES credentials requires
		// re-supplying the actual password, so PATCHing the mask back is rejected before
		// Fleet ever contacts the NDES server.
		res := patchNDES(fleet.NDESSCEPProxyCAUpdatePayload{
			Username: new("some-other-username"),
			Password: new(fleet.MaskedPassword),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "Invalid NDES SCEP password")

		require.Equal(t, before, authenticatedRequests.Load())
		requireStoredUsername(newNdesUsername) // saved by the previous subtest
	})
}

func (s *integrationMDMTestSuite) TestSCEPChallengeExpirationRetriesSmallStep() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Test setup
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// setup: create enroll secret, host, enroll to MDM
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)
	defaultProfiles := [][]byte{
		setupExpectedFleetdProfile(t, s.server.URL, t.Name(), nil),
		setupExpectedCAProfile(t, s.ds),
	}
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	installs, removes := checkNextPayloads(t, mdmDevice, false)
	s.signedProfilesMatch(
		defaultProfiles,
		installs,
	)
	require.Empty(t, removes)
	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	// setup: start smallstep scep server
	scepServer := scep_server.StartTestSCEPServer(t)

	// setup: start mock challenge server that returns new challenge value on each request
	challengeCounter := atomic.Int64{}
	challengeValue := atomic.Value{}
	challengeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that the request includes Authorization header with Basic auth
		authHeader := r.Header.Get("Authorization")
		require.NotEmpty(t, authHeader, "Authorization header should be present")
		require.Contains(t, authHeader, "Basic ", "Authorization header should use Basic auth")

		challengeCounter.Add(1)
		newChallengeValue := uuid.New().String()
		challengeValue.Store(newChallengeValue)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(newChallengeValue))
		require.NoError(t, err)
	}))
	t.Cleanup(func() {
		challengeServer.Close()
	})

	// setup: create smallstep CA in Fleet that uses the mock servers
	caName := "STEP_WIFI"
	_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{
		CertificateAuthorities: fleet.GroupedCertificateAuthorities{
			Smallstep: []fleet.SmallstepSCEPProxyCA{
				{
					Name:         caName,
					URL:          scepServer.URL + "/scep",
					ChallengeURL: challengeServer.URL,
					Username:     "testuser",
					Password:     "testpassword",
				},
			},
		},
		DryRun: false,
	}, http.StatusOK)
	require.NoError(t, err)
	require.Equal(t, int64(1), challengeCounter.Load()) // challenge endpoint called once during CA creation

	// setup: create a configuration profile that uses the smallstep CA for SCEP
	var profUUID string
	p := generateTestProfileSmallstepSCEP("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_STEP_WIFI", "$FLEET_VAR_SCEP_RENEWAL_ID", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_STEP_WIFI")
	body, headers := generateNewProfileMultipartRequest(t, "foobar.mobileconfig", []byte(p), s.token, nil)
	_ = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &profUUID, "SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE name = ?", "Smallstep Fleet WIFI")
	})

	// scepProfileURL is the expected SCEP profile URL after variable substitution (see preprocessProfileContents for details)
	scepProfileURL := fmt.Sprintf("%s%s%s", s.server.URL, apple_mdm.SCEPProxyPath,
		url.PathEscape(fmt.Sprintf("%s,%s,%s", host.UUID, profUUID, caName)))

	// expectPayloadWithChallenge executes the certificate profile template with current challenge value and other Fleet variables
	expectPayloadWithChallenge := func() string {
		challengeVal, ok := challengeValue.Load().(string)
		require.True(t, ok, "challenge value not set")
		return generateTestProfileSmallstepSCEP(
			challengeVal,
			"fleet-"+profUUID,
			scepProfileURL,
		)
	}

	// parseCommandPayload extracts and returns the profile payload from an InstallProfile command
	parseCommandPayload := func(cmd *mdm.Command) string {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		p7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
		require.NoError(t, err)
		return string(p7.Content)
	}

	// hostProfile represents the relevant fields from host_mdm_apple_profiles table for verification
	type hostProfile struct {
		ProfileUUID       string  `db:"profile_uuid"`
		ProfileIdentifier string  `db:"profile_identifier"`
		ProfileName       string  `db:"profile_name"`
		Status            *string `db:"status"`
		OperationType     *string `db:"operation_type"`
		Retries           int     `db:"retries"`
		CommandUUID       string  `db:"command_uuid"`
	}

	// listHostProfilesDB lists the host profiles for a given host from the database
	listHostProfilesDB := func(hostUUID string) []hostProfile {
		var got []hostProfile
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			// for the purpose of this test, we ignore the Fleet-internal profiles
			// (we only care about the custom profiles)
			return sqlx.SelectContext(t.Context(), q, &got, `
				SELECT profile_uuid, profile_identifier, profile_name, status, operation_type, retries, command_uuid
				FROM host_mdm_apple_profiles
				WHERE host_uuid = ? AND profile_identifier NOT IN (?, ?)`,
				hostUUID, mobileconfig.FleetdConfigPayloadIdentifier, mobileconfig.FleetCARootConfigPayloadIdentifier)
		})
		return got
	}

	// expectHostProf represents the expected host profile entry in the database; we'll update fields as we progress through the test
	expectHostProf := hostProfile{
		ProfileUUID:       profUUID,               // should never change
		ProfileIdentifier: "Smallstep Fleet WIFI", // should never change
		ProfileName:       "Smallstep Fleet WIFI", // should never change
		OperationType:     ptr.String("install"),  // should never change
		Status:            nil,                    // status is a key part of the test progression
		Retries:           0,                      // retries is a key part of the test progression
		CommandUUID:       "",                     // command UUID is a key part of the test progression
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Test scenarios
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	s.awaitTriggerProfileSchedule(t)
	require.Equal(t, int64(2), challengeCounter.Load()) // challenge endpoint called during host profile reconciliation

	// MDM checkin should expect InstallProfile command with SCEP profile
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	prevCommandUUID := cmd.CommandUUID // save for later comparison
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	// verify that the install profile command contains the expected payload, including the expected challenge
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd))

	// update expectations for host profile DB state
	expectHostProf.CommandUUID = cmd.CommandUUID
	expectHostProf.Status = ptr.String("pending")
	expectHostProf.Retries = 0 // initial install attempt

	// check DB state
	gotHostProfs := listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// simulate random failure during SCEP protocol
	cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
	require.NoError(t, err) // error report accepted by server
	require.Nil(t, cmd)     // no new command should be issued yet

	// update expectations for host profile DB state after failure
	expectHostProf.CommandUUID = prevCommandUUID // unchanged
	expectHostProf.Status = nil                  // status should be cleared to allow retry
	expectHostProf.Retries = 1                   // retries should be incremented

	// check DB state after failure
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// MDM checkin should not expect a new command yet
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// trigger another profile sync, which should resend SCEP profile
	require.Equal(t, int64(2), challengeCounter.Load()) // challenge endpoint not called until reconcilation runs
	s.awaitTriggerProfileSchedule(t)
	require.Equal(t, int64(3), challengeCounter.Load()) // challenge endpoint called with host profile reconciliation

	// MDM checkin should expect InstallProfile command with SCEP profile with new challenge
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotEqual(t, prevCommandUUID, cmd.CommandUUID) // new command UUID
	prevCommandUUID = cmd.CommandUUID                     // save for later comparison
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd)) // challenge value should be updated

	// update expectations for host profile DB state
	expectHostProf.CommandUUID = cmd.CommandUUID  // should be updated to new command UUID
	expectHostProf.Status = ptr.String("pending") // should now be pending again
	expectHostProf.Retries = 1                    // unchanged

	// check DB state
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// simulate additional failures during SCEP protocol until retry limit is exhausted.
	// Each error when retries < MaxAppleProfileRetries triggers a retry (retries++, status=NULL).
	// When retries == MaxAppleProfileRetries, the next error marks as failed.
	expectedChallengeCount := int64(3) // starts at 3 from above
	for retries := 1; retries < servermdm.MaxAppleProfileRetries; retries++ {
		// error triggers retry
		cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
		require.NoError(t, err)
		require.Nil(t, cmd)

		expectHostProf.CommandUUID = prevCommandUUID
		expectHostProf.Status = nil
		expectHostProf.Retries = retries + 1

		gotHostProfs = listHostProfilesDB(host.UUID)
		require.Len(t, gotHostProfs, 1)
		require.Equal(t, expectHostProf, gotHostProfs[0])

		// cron resends with new challenge
		s.awaitTriggerProfileSchedule(t)
		expectedChallengeCount++
		require.Equal(t, expectedChallengeCount, challengeCounter.Load())

		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.NotEqual(t, prevCommandUUID, cmd.CommandUUID)
		prevCommandUUID = cmd.CommandUUID
		require.Equal(t, "InstallProfile", cmd.Command.RequestType)

		expectHostProf.CommandUUID = cmd.CommandUUID
		expectHostProf.Status = ptr.String("pending")

		gotHostProfs = listHostProfilesDB(host.UUID)
		require.Len(t, gotHostProfs, 1)
		require.Equal(t, expectHostProf, gotHostProfs[0])
	}

	// final error: retries == MaxAppleProfileRetries, should mark as failed
	cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
	require.NoError(t, err)
	require.Nil(t, cmd)

	expectHostProf.CommandUUID = prevCommandUUID
	expectHostProf.Status = ptr.String("failed")
	expectHostProf.Retries = servermdm.MaxAppleProfileRetries

	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	s.awaitTriggerProfileSchedule(t)
	require.Equal(t, expectedChallengeCount, challengeCounter.Load())

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// manually resend the profile installation, which ignores retry limit
	// FIXME: manual resend doesn't change retries, but maybe it should reset to 0
	_ = s.Do("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, profUUID), nil, http.StatusAccepted)
	require.Equal(t, expectedChallengeCount, challengeCounter.Load()) // challenge endpoint not called until reconcilation runs

	// MDM checkin should not expect new command until reconciliation runs
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd) // no new command should be issued yet

	// update expectations for host profile DB state after manual resend
	expectHostProf.Status = nil                               // status should be cleared to allow retry
	expectHostProf.Retries = servermdm.MaxAppleProfileRetries // unchanged for manual resend
	expectHostProf.CommandUUID = prevCommandUUID              // unchanged until reconcilation runs

	// check DB state after manual resend request
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// trigger another profile sync, which should resend SCEP profile
	require.Equal(t, expectedChallengeCount, challengeCounter.Load()) // challenge endpoint not called until reconcilation runs
	s.awaitTriggerProfileSchedule(t)
	expectedChallengeCount++
	require.Equal(t, expectedChallengeCount, challengeCounter.Load()) // challenge endpoint called again during host profile reconciliation

	// MDM checkin should expect InstallProfile command with SCEP profile with new challenge
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotEqual(t, prevCommandUUID, cmd.CommandUUID) // new command UUID
	prevCommandUUID = cmd.CommandUUID                     // save for later comparison
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd)) // challenge value should be updated

	// update expectations for host profile DB state
	expectHostProf.CommandUUID = cmd.CommandUUID              // should be updated to new command UUID
	expectHostProf.Status = ptr.String("pending")             // should now be pending again
	expectHostProf.Retries = servermdm.MaxAppleProfileRetries // unchanged for manual resend

	// check DB state
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// simulate challenge expiration by backdating challenge_retrieved_at
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, _ = q.ExecContext(context.Background(), "UPDATE host_mdm_managed_certificates SET challenge_retrieved_at = DATE_SUB(challenge_retrieved_at, INTERVAL 270 SECOND) WHERE host_uuid = ?", host.UUID)
		return nil
	})

	// simulate MDM client sending SCEP request after challenge has expired
	resp, err := http.Get(scepProfileURL + "?operation=PKIOperation&message=" + base64.URLEncoding.EncodeToString([]byte("dummy")))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, extractServerErrorText(resp.Body), "challenge password has expired") // Fleet intercepts the SCEP request and returns an error

	// expired challenge should cause retries to be reset and command UUID cleared in DB
	expectHostProf.Status = nil     // status should be cleared to allow retry
	expectHostProf.Retries = 0      // retries should be reset to 0
	expectHostProf.CommandUUID = "" // command UUID should be cleared
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// MDM client reports error for the last InstallProfile command, which doesn't impact retries
	// because the command UUID was cleared when challenge expired
	cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
	require.NoError(t, err) // server accepted the error report
	require.Nil(t, cmd)     // no new command should be issued yet

	// reported error should not impact host profile DB state because command UUID was cleared when challenge expired
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// MDM checkin should not expect new command until reconciliation runs
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// trigger another profile sync, which should resend the SCEP profile installation
	require.Equal(t, expectedChallengeCount, challengeCounter.Load()) // challenge endpoint not called until reconcilation runs
	s.awaitTriggerProfileSchedule(t)
	expectedChallengeCount++
	require.Equal(t, expectedChallengeCount, challengeCounter.Load()) // challenge endpoint called with host profile reconciliation

	// MDM checkin should expect InstallProfile command with SCEP profile with new challenge
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	require.NotEqual(t, prevCommandUUID, cmd.CommandUUID)
	// prevCommandUUID = cmd.CommandUUID                                        // save for later comparison
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd)) // challenge value should be updated

	// verify that host profile DB state reflects new InstallProfile command
	expectHostProf.Status = ptr.String("pending") // should now be pending again
	expectHostProf.Retries = 0                    // unchanged
	expectHostProf.CommandUUID = cmd.CommandUUID  // should be updated to new command UUID
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])
}

// TestSCEPChallengeExpirationRetriesNDES is the NDES counterpart to
// TestSCEPChallengeExpirationRetriesSmallStep. It drives an NDES SCEP profile
// through the install/retry/resend lifecycle and, critically, exercises the
// expired-challenge path: when the MDM client performs a SCEP PKIOperation
// after the cached challenge has expired, Fleet should intercept it, resend the
// profile with a fresh challenge, and not let the resulting (stale) command
// failure corrupt the profile's DB state.
//
// This mirrors the Smallstep test: for an Apple profile, the expired-challenge
// branch in ee/server/service/scep/scep_proxy.go routes through
// ResendHostCertificateProfile, which clears command_uuid and resets the retry
// counter (an expired challenge is a timing condition, not a host install
// failure). The key invariants asserted below are flagged with "INVARIANT"
// comments.
func (s *integrationMDMTestSuite) TestSCEPChallengeExpirationRetriesNDES() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Test setup
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// setup: create enroll secret, host, enroll to MDM
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)
	defaultProfiles := [][]byte{
		setupExpectedFleetdProfile(t, s.server.URL, t.Name(), nil),
		setupExpectedCAProfile(t, s.ds),
	}
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	installs, removes := checkNextPayloads(t, mdmDevice, false)
	s.signedProfilesMatch(
		defaultProfiles,
		installs,
	)
	require.Empty(t, removes)
	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	// setup: start smallstep-backed SCEP server (the SCEP protocol server is
	// the same for NDES; only the challenge source differs)
	scepServer := scep_server.StartTestSCEPServer(t)

	// setup: start a mock NDES admin server that returns a fresh challenge
	// password on each request. NDES challenge passwords are single-use, so
	// every fetch must produce a new value.
	challengeCounter := atomic.Int64{}
	challengeValue := atomic.Value{}
	ndesAdminServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// NDES challenge passwords are single-use and rotate on each fetch.
		// (Note: the NDES admin URL is reached via an NTLM negotiator that may
		// send an unauthenticated probe first, so we don't assert on the
		// Authorization header here.)
		challengeCounter.Add(1)
		newChallengeValue := strings.ReplaceAll(uuid.New().String(), "-", "")
		challengeValue.Store(newChallengeValue)
		w.WriteHeader(http.StatusOK)
		// Fleet parses the challenge out of this exact HTML shape (see
		// GetNDESSCEPChallenge / challengeRegex).
		_, _ = w.Write([]byte(fmt.Sprintf(
			`<HTML><BODY>The enrollment challenge password is: <B> %s </B></BODY></HTML>`, newChallengeValue)))
	}))
	t.Cleanup(ndesAdminServer.Close)

	// setup: create the NDES CA in Fleet (singleton). Creating/validating it
	// fetches a challenge from the admin server.
	_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", batchApplyCertificateAuthoritiesRequest{
		CertificateAuthorities: fleet.GroupedCertificateAuthorities{
			NDESSCEP: &fleet.NDESSCEPProxyCA{
				URL:      scepServer.URL + "/scep",
				AdminURL: ndesAdminServer.URL + "/mscep_admin/",
				Username: "testuser",
				Password: "testpassword",
			},
		},
		DryRun: false,
	}, http.StatusOK)
	require.Positive(t, challengeCounter.Load(), "challenge endpoint should be called during CA validation")

	// setup: create a configuration profile that uses the NDES CA for SCEP
	var profUUID string
	p := generateTestProfileNDESSCEP("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_SCEP_RENEWAL_ID", "$FLEET_VAR_NDES_SCEP_PROXY_URL")
	body, headers := generateNewProfileMultipartRequest(t, "ndes.mobileconfig", []byte(p), s.token, nil)
	_ = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &profUUID, "SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE name = ?", "NDES Fleet WIFI")
	})

	// scepProfileURL is the expected SCEP proxy URL after variable substitution
	// (see BuildNDESSCEPProxyURL). NDES appends ",NDES" as the CA-name component.
	scepProfileURL := fmt.Sprintf("%s%s%s", s.server.URL, apple_mdm.SCEPProxyPath,
		url.PathEscape(fmt.Sprintf("%s,%s,NDES", host.UUID, profUUID)))

	// expectPayloadWithChallenge executes the profile template with the current
	// (most recently fetched) challenge value and the other substituted Fleet
	// variables.
	expectPayloadWithChallenge := func() string {
		challengeVal, ok := challengeValue.Load().(string)
		require.True(t, ok, "challenge value not set")
		return generateTestProfileNDESSCEP(
			challengeVal,
			"fleet-"+profUUID,
			scepProfileURL,
		)
	}

	// parseCommandPayload extracts the profile payload from an InstallProfile command
	parseCommandPayload := func(cmd *mdm.Command) string {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		p7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
		require.NoError(t, err)
		return string(p7.Content)
	}

	// hostProfile represents the relevant fields from host_mdm_apple_profiles for verification
	type hostProfile struct {
		ProfileUUID       string  `db:"profile_uuid"`
		ProfileIdentifier string  `db:"profile_identifier"`
		ProfileName       string  `db:"profile_name"`
		Status            *string `db:"status"`
		OperationType     *string `db:"operation_type"`
		Retries           int     `db:"retries"`
		CommandUUID       string  `db:"command_uuid"`
	}

	listHostProfilesDB := func(hostUUID string) []hostProfile {
		var got []hostProfile
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			// ignore the Fleet-internal profiles; we only care about the custom NDES profile
			return sqlx.SelectContext(t.Context(), q, &got, `
				SELECT profile_uuid, profile_identifier, profile_name, status, operation_type, retries, command_uuid
				FROM host_mdm_apple_profiles
				WHERE host_uuid = ? AND profile_identifier NOT IN (?, ?)`,
				hostUUID, mobileconfig.FleetdConfigPayloadIdentifier, mobileconfig.FleetCARootConfigPayloadIdentifier)
		})
		return got
	}

	// expectHostProf is the running expectation for the custom NDES profile's DB row
	expectHostProf := hostProfile{
		ProfileUUID:       profUUID,
		ProfileIdentifier: "NDES Fleet WIFI",
		ProfileName:       "NDES Fleet WIFI",
		OperationType:     new("install"),
		Status:            nil,
		Retries:           0,
		CommandUUID:       "",
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Test scenarios
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// reconcile fetches a fresh challenge and enqueues the InstallProfile command
	beforeReconcile := challengeCounter.Load()
	s.awaitTriggerProfileSchedule(t)
	require.Greater(t, challengeCounter.Load(), beforeReconcile, "challenge endpoint should be called during host profile reconciliation")

	// MDM checkin should deliver the InstallProfile command with the SCEP profile
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	prevCommandUUID := cmd.CommandUUID
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd))
	prevChallenge, _ := challengeValue.Load().(string)

	expectHostProf.CommandUUID = cmd.CommandUUID
	expectHostProf.Status = new("pending")
	expectHostProf.Retries = 0

	gotHostProfs := listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// Drive failures until the retry limit is exhausted. Each failure below the
	// limit triggers a resend with a brand-new challenge (the core behavior the
	// user asked about). When retries == MaxAppleProfileRetries, the next
	// failure marks the profile failed.
	for retries := range servermdm.MaxAppleProfileRetries {
		// device reports failure for the current install command
		cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
		require.NoError(t, err)
		require.Nil(t, cmd)

		expectHostProf.CommandUUID = prevCommandUUID
		expectHostProf.Status = nil
		expectHostProf.Retries = retries + 1
		gotHostProfs = listHostProfilesDB(host.UUID)
		require.Len(t, gotHostProfs, 1)
		require.Equal(t, expectHostProf, gotHostProfs[0])

		// cron resends with a new challenge
		s.awaitTriggerProfileSchedule(t)
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.NotEqual(t, prevCommandUUID, cmd.CommandUUID)
		require.Equal(t, "InstallProfile", cmd.Command.RequestType)
		// a fresh, different challenge should have been fetched and embedded
		newChallenge, _ := challengeValue.Load().(string)
		require.NotEqual(t, prevChallenge, newChallenge, "expected a new NDES challenge on resend")
		require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd))
		prevChallenge = newChallenge
		prevCommandUUID = cmd.CommandUUID

		expectHostProf.CommandUUID = cmd.CommandUUID
		expectHostProf.Status = new("pending")
		gotHostProfs = listHostProfilesDB(host.UUID)
		require.Len(t, gotHostProfs, 1)
		require.Equal(t, expectHostProf, gotHostProfs[0])
	}

	// final failure: retries == MaxAppleProfileRetries, profile is marked failed
	cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
	require.NoError(t, err)
	require.Nil(t, cmd)

	expectHostProf.CommandUUID = prevCommandUUID
	expectHostProf.Status = new("failed")
	expectHostProf.Retries = servermdm.MaxAppleProfileRetries
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// manually resend the profile (ignores retry limit). This brings status
	// back to pending so the host has a live, in-flight SCEP command we can
	// then expire — matching the real-world setup for the expiry bug.
	_ = s.Do("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, profUUID), nil, http.StatusAccepted)
	s.awaitTriggerProfileSchedule(t)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotEqual(t, prevCommandUUID, cmd.CommandUUID)
	prevCommandUUID = cmd.CommandUUID
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd))

	expectHostProf.CommandUUID = cmd.CommandUUID
	expectHostProf.Status = new("pending")
	expectHostProf.Retries = servermdm.MaxAppleProfileRetries // manual resend doesn't reset retries
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// simulate challenge expiration by backdating challenge_retrieved_at past
	// NDESChallengeInvalidAfter (57m). 2 hours is comfortably beyond that.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, execErr := q.ExecContext(t.Context(),
			"UPDATE host_mdm_managed_certificates SET challenge_retrieved_at = DATE_SUB(challenge_retrieved_at, INTERVAL 2 HOUR) WHERE host_uuid = ? AND profile_uuid = ?",
			host.UUID, profUUID)
		require.NoError(t, execErr)
		rows, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), rows, "expected to backdate exactly the NDES profile's managed certificate row")
		return nil
	})

	// MDM client performs a SCEP PKIOperation after the challenge has expired;
	// Fleet intercepts and returns an error rather than forwarding it.
	resp, err := http.Get(scepProfileURL + "?operation=PKIOperation&message=" + base64.URLEncoding.EncodeToString([]byte("dummy")))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, extractServerErrorText(resp.Body), "challenge password has expired")

	// INVARIANT: the expired challenge resends via ResendHostCertificateProfile,
	// which clears command_uuid and resets retries so the resend is clean and
	// unbounded (status back to NULL).
	expectHostProf.Status = nil
	expectHostProf.Retries = 0
	expectHostProf.CommandUUID = ""
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// The MDM client now reports the failure for the install command that was
	// in flight when the challenge expired. Because the command_uuid should
	// have been cleared above, this stale failure must be a no-op and must NOT
	// corrupt the profile state.
	cmd, err = mdmDevice.Err(prevCommandUUID, []mdm.ErrorChain{})
	require.NoError(t, err)
	require.Nil(t, cmd)

	// INVARIANT: because command_uuid was cleared above, this stale failure ACK
	// matches no row and is a no-op — it must not flip the profile to "failed"
	// or otherwise change the DB state, so the resend can still proceed.
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])

	// reconcile should resend the profile with a fresh challenge
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd) // nothing until reconcile runs
	s.awaitTriggerProfileSchedule(t)

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "expected the SCEP profile to be resent after challenge expiration")
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	require.NotEqual(t, prevCommandUUID, cmd.CommandUUID)
	require.Equal(t, expectPayloadWithChallenge(), parseCommandPayload(cmd))

	expectHostProf.Status = new("pending")
	expectHostProf.Retries = 0
	expectHostProf.CommandUUID = cmd.CommandUUID
	gotHostProfs = listHostProfilesDB(host.UUID)
	require.Len(t, gotHostProfs, 1)
	require.Equal(t, expectHostProf, gotHostProfs[0])
}

func generateTestProfileNDESSCEP(challenge, ou, url string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
       <dict>
          <key>PayloadContent</key>
          <dict>
             <key>Challenge</key>
             <string>%s</string>
             <key>Key Type</key>
             <string>RSA</string>
             <key>Key Usage</key>
             <integer>5</integer>
             <key>Keysize</key>
             <integer>2048</integer>
             <key>Subject</key>
                    <array>
                        <array>
                          <array>
                            <string>CN</string>
                            <string>SerialNumber WIFI</string>
                          </array>
                        </array>
                        <array>
                          <array>
                            <string>OU</string>
                            <string>%s</string>
                          </array>
                        </array>
                    </array>
             <key>URL</key>
             <string>%s</string>
          </dict>
          <key>PayloadDisplayName</key>
          <string>WIFI SCEP</string>
          <key>PayloadIdentifier</key>
          <string>com.apple.security.scep.8ACC34A5-72F9-42B7-9A98-7AD9A9CCA3AF</string>
          <key>PayloadType</key>
          <string>com.apple.security.scep</string>
          <key>PayloadUUID</key>
          <string>8ACC34A5-72F9-42B7-9A98-7AD9A9CCA3AF</string>
          <key>PayloadVersion</key>
          <integer>1</integer>
       </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>NDES Fleet WIFI</string>
    <key>PayloadIdentifier</key>
    <string>NDES Fleet WIFI</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>3BD1BD65-1D2C-4E9E-9E18-9BCD400CDEDF</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`, challenge, ou, url)
}

func generateTestProfileSmallstepSCEP(challenge, ou, url string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
       <dict>
          <key>PayloadContent</key>
          <dict>
             <key>Challenge</key>
             <string>%s</string>
             <key>Key Type</key>
             <string>RSA</string>
             <key>Key Usage</key>
             <integer>5</integer>
             <key>Keysize</key>
             <integer>2048</integer>
             <key>Subject</key>
                    <array>
                        <array>
                          <array>
                            <string>CN</string>
                            <string>SerialNumber WIFI</string>
                          </array>
                        </array>
                        <array>
                          <array>
                            <string>OU</string>
                            <string>%s</string>
                          </array>
                        </array>
                    </array>
             <key>URL</key>
             <string>%s</string>
          </dict>
          <key>PayloadDisplayName</key>
          <string>WIFI SCEP</string>
          <key>PayloadIdentifier</key>
          <string>com.apple.security.scep.9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AE</string>
          <key>PayloadType</key>
          <string>com.apple.security.scep</string>
          <key>PayloadUUID</key>
          <string>9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AE</string>
          <key>PayloadVersion</key>
          <integer>1</integer>
       </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Smallstep Fleet WIFI</string>
    <key>PayloadIdentifier</key>
    <string>Smallstep Fleet WIFI</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>4CD1BD65-1D2C-4E9E-9E18-9BCD400CDEDE</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`, challenge, ou, url)
}

func (s *integrationMDMTestSuite) TestEJBCACertificateAuthorities() {
	t := s.T()
	ctx := context.Background()
	deleteCAs := func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "DELETE FROM certificate_authorities")
			return err
		})
	}
	deleteCAs()
	t.Cleanup(deleteCAs)

	fake := ejbcatest.New(t, func(f *ejbcatest.Server) { f.CertProfiles = []string{ejbcatest.CertProfile, "otherProfile"} })
	const p12Password = "p12-password"
	expiry := time.Now().Add(90 * 24 * time.Hour).UTC().Truncate(time.Second)
	untrusted := ejbcatest.NewCA(t, "Other CA")
	createReq := func(name, p12 string) map[string]any {
		return map[string]any{"ejbca": map[string]any{
			"name":                             name,
			"url":                              fake.HTTP.URL,
			"client_p12":                       p12,
			"password":                         p12Password,
			"trust_ca_bundle":                  fake.ServerPEM,
			"certificate_authority_name":       ejbcatest.CAName,
			"certificate_profile_name":         ejbcatest.CertProfile,
			"end_entity_profile_name":          ejbcatest.EndEntityProfile,
			"certificate_common_name":          "$FLEET_VAR_HOST_END_USER_EMAIL_IDP",
			"username_template":                "$FLEET_VAR_HOST_HARDWARE_SERIAL",
			"certificate_user_principal_names": []string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP"},
		}}
	}
	trusted := fake.ClientP12(t, expiry, p12Password)

	// A client certificate EJBCA doesn't trust, and names it doesn't know, are rejected and not stored.
	res := s.Do("POST", "/api/latest/fleet/certificate_authorities",
		createReq("EJBCA_REJECTED", ejbcatest.ClientP12(t, untrusted, expiry, p12Password)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "EJBCA rejected the TLS client certificate")
	unknownCA := createReq("EJBCA_UNKNOWN_CA", trusted)
	unknownCA["ejbca"].(map[string]any)["certificate_authority_name"] = "NoSuchCA"
	res = s.Do("POST", "/api/latest/fleet/certificate_authorities", unknownCA, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `EJBCA CA "NoSuchCA" was not found`)
	unknownProfile := createReq("EJBCA_UNKNOWN_PROFILE", trusted)
	unknownProfile["ejbca"].(map[string]any)["certificate_profile_name"] = "noSuchProfile"
	res = s.Do("POST", "/api/latest/fleet/certificate_authorities", unknownProfile, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `certificate profile "noSuchProfile" was not found`)
	var listResp listCertificateAuthoritiesResponse
	s.DoJSON("GET", "/api/latest/fleet/certificate_authorities", nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.CertificateAuthorities)

	// Create, and the same name is already used.
	var createResp createCertificateAuthorityResponse
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", createReq("EJBCA_WIFI", trusted), http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	require.Equal(t, fleet.CATypeEJBCA, createResp.Type)
	s.lastActivityMatches(fleet.ActivityAddedEJBCA{}.ActivityName(), `{"name":"EJBCA_WIFI"}`, 0)
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", createReq("EJBCA_WIFI", trusted), http.StatusBadRequest,
		&createCertificateAuthorityResponse{})

	// No response has a secret: not the PKCS#12 or its password, and not the stored key, in PEM or as the base64 of its body.
	stored, err := s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, true)
	require.NoError(t, err)
	keyBody := strings.NewReplacer("-----BEGIN PRIVATE KEY-----", "", "-----END PRIVATE KEY-----", "", "\n", "").Replace(*stored.MTLSKeyPEM)
	require.NotEmpty(t, keyBody)
	requireNoSecrets := func(body []byte) {
		for _, secret := range []string{"PRIVATE KEY", "client_p12", p12Password, *stored.MTLSKeyPEM, keyBody} {
			require.NotContains(t, string(body), secret)
		}
	}

	// The list has the client certificate expiry.
	listRes := s.DoRaw("GET", "/api/latest/fleet/certificate_authorities", nil, http.StatusOK)
	body, err := io.ReadAll(listRes.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &listResp))
	require.Len(t, listResp.CertificateAuthorities, 1)
	require.NotNil(t, listResp.CertificateAuthorities[0].MTLSCertExpiresAt)
	require.True(t, expiry.Equal(*listResp.CertificateAuthorities[0].MTLSCertExpiresAt))
	requireNoSecrets(body)

	// A get has the settings and the expiry.
	var getResp getCertificateAuthorityResponse
	getRes := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", createResp.ID), nil, http.StatusOK)
	body, err = io.ReadAll(getRes.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &getResp))
	require.Equal(t, ejbcatest.CertProfile, *getResp.EJBCACertificateProfileName)
	require.Equal(t, "$FLEET_VAR_HOST_HARDWARE_SERIAL", *getResp.EJBCAUsernameTemplate)
	require.True(t, expiry.Equal(*getResp.MTLSCertExpiresAt))
	requireNoSecrets(body)

	// Editing a template doesn't contact EJBCA and keeps the stored credential.
	editURL := fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", createResp.ID)
	requests := len(fake.Requests())
	s.Do("PATCH", editURL, map[string]any{"ejbca": map[string]any{"certificate_common_name": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}}, http.StatusOK)
	s.lastActivityMatches(fleet.ActivityEditedEJBCA{}.ActivityName(), `{"name":"EJBCA_WIFI"}`, 0)
	require.Len(t, fake.Requests(), requests)
	stored, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, true)
	require.NoError(t, err)
	require.Equal(t, "$FLEET_VAR_HOST_HARDWARE_SERIAL", *stored.CertificateCommonName)
	require.Contains(t, *stored.MTLSKeyPEM, "PRIVATE KEY")

	// Editing the certificate profile verifies it with EJBCA using the stored credential, and an unknown profile is rejected.
	s.Do("PATCH", editURL, map[string]any{"ejbca": map[string]any{"certificate_profile_name": "otherProfile"}}, http.StatusOK)
	require.Equal(t, []string{
		"GET " + ejbcatest.RESTAPIPrefix + "/v1/ca",
		"GET " + ejbcatest.RESTAPIPrefix + "/v2/certificate/profile/otherProfile",
	}, fake.Requests()[requests:])
	res = s.Do("PATCH", editURL, map[string]any{"ejbca": map[string]any{"certificate_profile_name": "noSuchProfile"}}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `certificate profile "noSuchProfile" was not found`)
	got, err := s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, false)
	require.NoError(t, err)
	require.Equal(t, "otherProfile", *got.EJBCACertificateProfileName)

	// A replacement credential EJBCA rejects leaves the stored one in place, and one it accepts replaces it.
	newExpiry := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second)
	s.Do("PATCH", editURL, map[string]any{"ejbca": map[string]any{
		"client_p12": ejbcatest.ClientP12(t, untrusted, newExpiry, p12Password), "password": p12Password,
	}}, http.StatusBadRequest)
	got, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, false)
	require.NoError(t, err)
	require.True(t, expiry.Equal(*got.MTLSCertExpiresAt))
	rotated := map[string]any{"client_p12": fake.ClientP12(t, newExpiry, p12Password), "password": p12Password}
	s.Do("PATCH", editURL, map[string]any{"ejbca": rotated}, http.StatusOK)
	got, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, false)
	require.NoError(t, err)
	require.True(t, newExpiry.Equal(*got.MTLSCertExpiresAt))

	// Delete.
	s.Do("DELETE", editURL, nil, http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityDeletedEJBCA{}.ActivityName(), `{"name":"EJBCA_WIFI"}`, 0)
	_, err = s.ds.GetCertificateAuthorityByID(ctx, createResp.ID, false)
	require.Error(t, err)
}

func (s *integrationMDMTestSuite) TestEJBCACertificateAuthorityNameCannotBeShared() {
	t := s.T()
	ctx := t.Context()
	// A name-only edit doesn't contact EJBCA, so the CAs can be set up directly, with the fields an edit validates.
	ejbca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type: string(fleet.CATypeEJBCA), Name: new("EJBCA_WIFI"), URL: new("https://ejbca.example.com"), EJBCACAName: new("CA"),
		EJBCACertificateProfileName: new("profile"), EJBCAEndEntityProfileName: new("entity"), CertificateCommonName: new("cn"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = s.ds.DeleteCertificateAuthority(context.Background(), ejbca.ID) })
	scep, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type: string(fleet.CATypeCustomSCEPProxy), Name: new("SCEP_WIFI"), URL: new("https://scep.example.com"), Challenge: new("c"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = s.ds.DeleteCertificateAuthority(context.Background(), scep.ID) })

	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ejbca.ID),
		map[string]any{"ejbca": map[string]any{"name": "SCEP_WIFI"}}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `"SCEP_WIFI" name is already used by another certificate authority`)
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", scep.ID),
		map[string]any{"custom_scep_proxy": map[string]any{"name": "EJBCA_WIFI"}}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `"EJBCA_WIFI" name is already used by another certificate authority`)
}

func (s *integrationMDMTestSuite) TestBatchApplyEJBCACertificateAuthorities() {
	t := s.T()
	ctx := t.Context()
	deleteCAs := func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "DELETE FROM certificate_authorities")
			return err
		})
	}
	deleteCAs()
	t.Cleanup(deleteCAs)

	fake := ejbcatest.New(t, nil)
	scepServer := scep_server.StartTestSCEPServer(t)
	t.Cleanup(scepServer.Close)
	const p12Password = "p12-password"
	expiry := time.Now().Add(90 * 24 * time.Hour).UTC().Truncate(time.Second)
	ejbcaSpec := func(name, p12 string) fleet.EJBCA {
		return fleet.EJBCA{
			Name: name, URL: fake.HTTP.URL, ClientP12: p12, Password: p12Password, TrustCABundlePEM: fake.ServerPEM,
			CertificateAuthorityName: ejbcatest.CAName, CertificateProfileName: ejbcatest.CertProfile,
			EndEntityProfileName: ejbcatest.EndEntityProfile, CertificateCommonName: "$FLEET_VAR_HOST_END_USER_EMAIL_IDP",
			UsernameTemplate: "$FLEET_VAR_HOST_HARDWARE_SERIAL", CertificateUserPrincipalNames: []string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP"},
		}
	}
	scepSpec := func(name string) fleet.CustomSCEPProxyCA {
		return fleet.CustomSCEPProxyCA{Name: name, URL: scepServer.URL + "/scep", Challenge: "challenge"}
	}
	// applyCAs posts the spec and returns the error text, if the status is not a success.
	applyCAs := func(req batchApplyCertificateAuthoritiesRequest, status int) string {
		res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, status)
		if status == http.StatusOK {
			return ""
		}
		return extractServerErrorText(res.Body)
	}
	apply := func(dryRun bool, status int, ejbcas ...fleet.EJBCA) string {
		return applyCAs(batchApplyCertificateAuthoritiesRequest{
			CertificateAuthorities: fleet.GroupedCertificateAuthorities{EJBCA: ejbcas}, DryRun: dryRun,
		}, status)
	}
	stored := func() []fleet.EJBCA {
		grouped, err := s.ds.GetGroupedCertificateAuthorities(ctx, true)
		require.NoError(t, err)
		return grouped.EJBCA
	}
	caNames := func() []string {
		cas, err := s.ds.ListCertificateAuthorities(ctx)
		require.NoError(t, err)
		names := make([]string, 0, len(cas))
		for _, ca := range cas {
			names = append(names, ca.Type+":"+ca.Name)
		}
		return names
	}

	wifi := ejbcaSpec("EJBCA_WIFI", fake.ClientP12(t, expiry, p12Password))

	// A dry run validates against EJBCA but stores nothing.
	apply(true, http.StatusOK, wifi)
	require.Empty(t, stored())
	require.NotEmpty(t, fake.Requests())

	// Add.
	apply(false, http.StatusOK, wifi)
	got := stored()
	require.Len(t, got, 1)
	require.Equal(t, wifi.CertificateUserPrincipalNames, got[0].CertificateUserPrincipalNames)
	require.Contains(t, got[0].MTLSCertPEM, "BEGIN CERTIFICATE")
	require.Contains(t, got[0].MTLSKeyPEM, "PRIVATE KEY")
	added := s.lastActivityMatches(fleet.ActivityAddedEJBCA{}.ActivityName(), `{"name":"EJBCA_WIFI"}`, 0)

	// The same spec changes nothing and doesn't contact EJBCA.
	requests := len(fake.Requests())
	apply(false, http.StatusOK, wifi)
	require.Len(t, fake.Requests(), requests)
	s.lastActivityMatches(fleet.ActivityAddedEJBCA{}.ActivityName(), "", added)

	// A changed field is validated again and stored.
	wifi.CertificateCommonName = "new-common-name"
	apply(false, http.StatusOK, wifi)
	require.Greater(t, len(fake.Requests()), requests)
	require.Equal(t, "new-common-name", stored()[0].CertificateCommonName)
	edited := s.lastActivityMatches(fleet.ActivityEditedEJBCA{}.ActivityName(), `{"name":"EJBCA_WIFI"}`, 0)

	// A new client certificate replaces the stored one, and is an edit too.
	oldCert := stored()[0].MTLSCertPEM
	wifi.ClientP12 = fake.ClientP12(t, expiry, p12Password)
	apply(false, http.StatusOK, wifi)
	storedCert := stored()[0].MTLSCertPEM
	require.NotEqual(t, oldCert, storedCert)
	require.NotEqual(t, edited, s.lastActivityMatches(fleet.ActivityEditedEJBCA{}.ActivityName(), `{"name":"EJBCA_WIFI"}`, 0))

	// Bad credentials reject the whole batch, and the stored CA keeps its certificate.
	wrongPassword := wifi
	wrongPassword.Password = "wrong"
	require.Contains(t, apply(false, http.StatusUnprocessableEntity, wrongPassword), "certificate_authorities.ejbca: ")
	require.Equal(t, storedCert, stored()[0].MTLSCertPEM)
	expired := ejbcaSpec("EJBCA_EXPIRED", ejbcatest.ClientP12Between(t, fake.ClientCA, time.Now().Add(-48*time.Hour),
		time.Now().Add(-24*time.Hour), p12Password))
	require.Contains(t, apply(false, http.StatusUnprocessableEntity, wifi, expired), "expired")
	require.Equal(t, storedCert, stored()[0].MTLSCertPEM)
	require.Len(t, stored(), 1)

	// Names are unique among the EJBCA CAs, ignoring case like the database does, and an EJBCA name can't be shared with a CA of another
	// type. These fail in a dry run too.
	lowerCase := wifi
	lowerCase.Name = "ejbca_wifi"
	require.Contains(t, apply(false, http.StatusUnprocessableEntity, wifi, wifi), "already")
	require.Contains(t, apply(false, http.StatusUnprocessableEntity, wifi, lowerCase), "already")
	for _, tc := range []struct {
		name          string
		scepName      string
		googleCASName string
	}{
		{"custom SCEP", "EJBCA_WIFI", ""},
		{"custom SCEP in another case", "ejbca_wifi", ""},
		{"Google CAS", "", "EJBCA_WIFI"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grouped := fleet.GroupedCertificateAuthorities{EJBCA: []fleet.EJBCA{wifi}}
			if tc.scepName != "" {
				grouped.CustomScepProxy = []fleet.CustomSCEPProxyCA{scepSpec(tc.scepName)}
			}
			if tc.googleCASName != "" {
				grouped.GoogleCAS = []fleet.GoogleCASCA{{
					Name: tc.googleCASName, Project: "project", Location: "us-central1", CAPool: "pool", APIKeyJSON: `{"type":"service_account"}`,
				}}
			}
			errText := applyCAs(batchApplyCertificateAuthoritiesRequest{CertificateAuthorities: grouped, DryRun: true}, http.StatusBadRequest)
			require.Contains(t, errText, `"EJBCA_WIFI" name is already used by another certificate authority`)
		})
	}
	// The name of the NDES CA is reserved in any case, whether or not an NDES CA is configured.
	for _, name := range []string{"NDES", "ndes"} {
		require.Contains(t, apply(true, http.StatusUnprocessableEntity, ejbcaSpec(name, wifi.ClientP12)), "CA name cannot be NDES")
	}

	// A rename that only changes the case updates the CA, it doesn't delete it.
	apply(false, http.StatusOK, lowerCase)
	require.Equal(t, []string{"ejbca:ejbca_wifi"}, caNames())

	// GitOps applies the spec without deletes first and deletes later, after it cleans up the certificate templates. A CA that changes type
	// and keeps its name has to be replaced in the first pass anyway, because the name can't be shared with an EJBCA CA, while the other
	// CAs that aren't in the spec stay for the later pass.
	_, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type: string(fleet.CATypeCustomSCEPProxy), Name: new("NOT_IN_SPEC"), URL: new(scepServer.URL + "/scep"), Challenge: new("challenge"),
	})
	require.NoError(t, err)
	applyCAs(batchApplyCertificateAuthoritiesRequest{
		CertificateAuthorities: fleet.GroupedCertificateAuthorities{CustomScepProxy: []fleet.CustomSCEPProxyCA{scepSpec("ejbca_wifi")}},
		SkipDeletes:            true,
	}, http.StatusOK)
	require.ElementsMatch(t, []string{"custom_scep_proxy:ejbca_wifi", "custom_scep_proxy:NOT_IN_SPEC"}, caNames())
	s.lastActivityOfTypeMatches(fleet.ActivityDeletedEJBCA{}.ActivityName(), `{"name":"ejbca_wifi"}`, 0)
	applyCAs(batchApplyCertificateAuthoritiesRequest{
		CertificateAuthorities: fleet.GroupedCertificateAuthorities{EJBCA: []fleet.EJBCA{lowerCase}}, SkipDeletes: true,
	}, http.StatusOK)
	require.ElementsMatch(t, []string{"ejbca:ejbca_wifi", "custom_scep_proxy:NOT_IN_SPEC"}, caNames())

	// Leaving the CAs out of the spec deletes them.
	apply(false, http.StatusOK)
	require.Empty(t, caNames())
}
