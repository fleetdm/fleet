package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/jmoiron/sqlx"
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

	ndesSCEPServer := scep.NewTestSCEPServer(t)
	ndesAdminServer := scep.NewTestNDESAdminServer(t, "mscep_admin_password", http.StatusOK)
	dynamicChallengeServer := scep.NewTestDynamicChallengeServer(t)

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

	// goodSmallstepCA is a base object for testing with a valid Smallstep SCEP CA. Copy it to override specific fields in tests.
	goodSmallstepCA := fleet.SmallstepSCEPProxyCA{
		Name:         "VALID_SMALLSTEP_SCEP",
		URL:          mockSCEPServer.URL + "/scep",
		ChallengeURL: dynamicChallengeServer.URL + "/challenge",
		Username:     "user",
		Password:     "password",
	}

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
			require.Contains(t, errMsg, "Invalid SCEP URL")
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
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", req, http.StatusOK)
			s.checkAppliedCAs(t, s.ds, req.CertificateAuthorities)

			t.Cleanup(func() {
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
					Smallstep:       []fleet.SmallstepSCEPProxyCA{goodSmallstepCA},
				},
				DryRun: false,
			}
			s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", duplicateReq, http.StatusOK)
		})

		// TODO(hca): hydrant happy path and other specific tests
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

	if expectedCAs.NDESSCEP != nil {
		assert.NotNil(t, gotCAs.NDESSCEP)
		gotCAs.NDESSCEP.ID = 0 // ignore ID when comparing
		assert.Equal(t, expectedCAs.NDESSCEP, gotCAs.NDESSCEP)
	} else {
		assert.Empty(t, gotCAs.NDESSCEP)
	}
}
