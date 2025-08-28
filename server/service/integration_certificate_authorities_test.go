package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
)

func (s *integrationMDMTestSuite) TestBatchApplyCertificateAuthorities() {
	t := s.T()

	ndesSCEPServer := eeservice.NewTestSCEPServer(t)

	// TODO(hca); add mechanism to check that validation endpoints were called
	ndesAdminServer := eeservice.NewTestNDESAdminServer(t, "mscep_admin_password", http.StatusOK)

	scepURL := ndesSCEPServer.URL + "/scep"
	adminURL := ndesAdminServer.URL + "/mscep_admin/"
	username := "user"
	password := "password"

	makeTestPayload := func(ndes *fleet.NDESSCEPProxyCA, dryRun bool) applyCertificateAuthoritiesSpecRequest {
		return applyCertificateAuthoritiesSpecRequest{
			CertificateAuthorities: fleet.GroupedCertificateAuthorities{
				NDESSCEP: ndes,
			},
			DryRun: dryRun,
		}
	}

	type ndesCases struct {
		name       string
		payload    interface{}
		status     int
		errMessage string
	}

	for _, tc := range []ndesCases{
		{
			name: "invalid SCEP URL",
			payload: makeTestPayload(&fleet.NDESSCEPProxyCA{
				URL:      "://invalid-url",
				AdminURL: adminURL,
				Username: username,
				Password: password,
			}, false),
			status:     http.StatusUnprocessableEntity,
			errMessage: "Invalid NDES SCEP URL",
		},
		{
			name: "wrong SCEP URL",
			payload: makeTestPayload(&fleet.NDESSCEPProxyCA{
				URL:      "https://new2.com/mscep/mscep.dll",
				AdminURL: adminURL,
				Username: username,
				Password: password,
			}, false),
			status:     http.StatusBadRequest,
			errMessage: "Invalid SCEP URL",
		},
		{
			name: "empty password",
			payload: makeTestPayload(&fleet.NDESSCEPProxyCA{
				URL:      scepURL,
				AdminURL: adminURL,
				Username: username,
				// Password omitted, should fail validation
			}, false),
			status:     http.StatusUnprocessableEntity,
			errMessage: "NDES SCEP password cannot be empty.",
		},
		{
			name: "happy path",
			payload: makeTestPayload(&fleet.NDESSCEPProxyCA{
				URL:      scepURL,
				AdminURL: adminURL,
				Username: username,
				Password: password,
			}, false),
			status:     http.StatusOK,
			errMessage: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// confirm no existing CAs
			cas, err := s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
			require.NoError(t, err)
			require.Nil(t, cas.NDESSCEP)

			// apply test case
			res := s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", tc.payload, tc.status)
			if tc.errMessage != "" {
				errMsg := extractServerErrorText(res.Body)
				require.Contains(t, errMsg, tc.errMessage)
			} else {
				require.Equal(t, http.StatusOK, res.StatusCode)
				// confirm the expected payload was applied
				p, ok := tc.payload.(applyCertificateAuthoritiesSpecRequest)
				require.True(t, ok)
				cas, err = s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
				require.NoError(t, err)
				require.NotNil(t, cas.NDESSCEP)
				require.NotZero(t, cas.NDESSCEP.ID)
				require.Equal(t, *p.CertificateAuthorities.NDESSCEP, fleet.NDESSCEPProxyCA{
					URL:      cas.NDESSCEP.URL,
					AdminURL: cas.NDESSCEP.AdminURL,
					Username: cas.NDESSCEP.Username,
					Password: cas.NDESSCEP.Password,
				})
			}
		})

		// cleanup after each sub-test
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `DELETE FROM certificate_authorities WHERE (name, type) IN (('NDES', 'ndes_scep_proxy'))`
			_, err := q.ExecContext(context.Background(), stmt)
			return err
		})
	}

	for _, tc := range []ndesCases{
		{
			name:    "nil NDES deletes existing",
			payload: makeTestPayload(nil, false),
			status:  http.StatusOK,
		},
		{
			name:    "empty NDES deletes existing",
			payload: makeTestPayload(&fleet.NDESSCEPProxyCA{}, false),
			status:  http.StatusOK,
		},
		{
			name:    "dry run does not delete existing",
			payload: makeTestPayload(nil, true),
			status:  http.StatusOK,
		},
		{
			name:    "json null NDES deletes existing",
			payload: map[string]interface{}{"certificate_authorities": map[string]interface{}{"ndes_scep_proxy": nil}, "dry_run": false},
			status:  http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// confirm no existing CAs
			cas, err := s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
			require.NoError(t, err)
			require.Nil(t, cas.NDESSCEP)

			// create a CA to delete
			existingCA := fleet.NDESSCEPProxyCA{
				URL:      scepURL,
				AdminURL: adminURL,
				Username: username,
				Password: password,
			}
			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", makeTestPayload(&existingCA, false), http.StatusOK)

			// confirm it exists
			cas, err = s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
			require.NoError(t, err)
			require.NotNil(t, cas.NDESSCEP)
			require.NotZero(t, cas.NDESSCEP.ID)
			require.Equal(t, existingCA, fleet.NDESSCEPProxyCA{
				URL:      cas.NDESSCEP.URL,
				AdminURL: cas.NDESSCEP.AdminURL,
				Username: cas.NDESSCEP.Username,
				Password: cas.NDESSCEP.Password,
			})

			// mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			// 	mysql.DumpTable(t, q, "certificate_authorities")
			// 	return nil
			// })
			// mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			// 	stmt := `DELETE FROM certificate_authorities WHERE (name, type) IN (('NDES', 'ndes_scep_proxy'))`
			// 	r, err := q.ExecContext(context.Background(), stmt)
			// 	if err != nil {
			// 		return err
			// 	}
			// 	rows, _ := r.RowsAffected()
			// 	t.Logf("deleted %d rows from certificate_authorities", rows)
			// 	return err
			// })
			// mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			// 	mysql.DumpTable(t, q, "certificate_authorities")
			// 	return nil
			// })
			// now delete it

			// b, err := json.Marshal(tc.payload)
			// require.NoError(t, err)
			// fmt.Println("payload:", string(b))

			_ = s.Do("POST", "/api/v1/fleet/spec/certificate_authorities", tc.payload, tc.status)

			// confirm expected results basd on dry run or not
			p, ok := tc.payload.(applyCertificateAuthoritiesSpecRequest)
			if !ok {
				// try map form
				m, mok := tc.payload.(map[string]interface{})
				require.True(t, mok)
				b, err := json.Marshal(m)
				require.NoError(t, err)
				err = json.Unmarshal(b, &p)
				require.NoError(t, err)
			}

			if p.DryRun {
				// confirm it was not deleted
				cas, err = s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
				require.NoError(t, err)
				require.NotNil(t, cas.NDESSCEP)
			} else {
				// confirm it was deleted
				cas, err = s.ds.GetGroupedCertificateAuthorities(context.Background(), true)
				require.NoError(t, err)
				require.Nil(t, cas.NDESSCEP)

				// TODO(hca): check delete again is no-op (i.e. no activity created)?
			}
		})

		// cleanup after each sub-test
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `DELETE FROM certificate_authorities WHERE (name, type) IN (('NDES', 'ndes_scep_proxy'))`
			_, err := q.ExecContext(context.Background(), stmt)
			return err
		})
	}
}

// TODO(hca): test dry runs?

// TODO(hca): test update existing ndes

// TODO(hca): test activities once implemented

// TODO(hca): test no external validation if no changes

// 	TODO(hca) test cannot configure NDES without private key?

// func TestAppConfigCAs(t *testing.T) {
// 	t.Parallel()

// 	pathRegex := regexp.MustCompile(`^/mpki/api/v2/profile/([a-zA-Z0-9_-]+)$`)
// 	mockDigiCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodGet {
// 			w.WriteHeader(http.StatusMethodNotAllowed)
// 			return
// 		}

// 		matches := pathRegex.FindStringSubmatch(r.URL.Path)
// 		if len(matches) != 2 {
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}
// 		profileID := matches[1]

// 		resp := map[string]string{
// 			"id":     profileID,
// 			"name":   "Test CA",
// 			"status": "Active",
// 		}
// 		w.Header().Set("Content-Type", "application/json")
// 		err := json.NewEncoder(w).Encode(resp)
// 		require.NoError(t, err)
// 	}))
// 	defer mockDigiCertServer.Close()

// 	setUpDigiCert := func() configCASuite {
// 		mt := configCASuite{
// 			ctx:          license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium}),
// 			invalid:      &fleet.InvalidArgumentError{},
// 			newAppConfig: getAppConfigWithDigiCertIntegration(mockDigiCertServer.URL, "WIFI"),
// 			oldAppConfig: &fleet.AppConfig{},
// 			appConfig:    &fleet.AppConfig{},
// 			svc:          &Service{logger: log.NewLogfmtLogger(os.Stdout)},
// 		}
// 		mt.svc.config.Server.PrivateKey = "exists"
// 		mt.svc.digiCertService = digicert.NewService()
// 		addMockDatastoreForCA(t, mt)
// 		return mt
// 	}
// 	setUpCustomSCEP := func() configCASuite {
// 		mt := configCASuite{
// 			ctx:          license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium}),
// 			invalid:      &fleet.InvalidArgumentError{},
// 			newAppConfig: getAppConfigWithSCEPIntegration("https://example.com", "SCEP_WIFI"),
// 			oldAppConfig: &fleet.AppConfig{},
// 			appConfig:    &fleet.AppConfig{},
// 			svc:          &Service{logger: log.NewLogfmtLogger(os.Stdout)},
// 		}
// 		mt.svc.config.Server.PrivateKey = "exists"
// 		scepConfig := &scep_mock.SCEPConfigService{}
// 		scepConfig.ValidateSCEPURLFunc = func(_ context.Context, _ string) error { return nil }
// 		mt.svc.scepConfigService = scepConfig
// 		addMockDatastoreForCA(t, mt)
// 		return mt
// 	}

// 	t.Run("free license", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.ctx = license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree})
// 		mt.newAppConfig = &fleet.AppConfig{}
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.ndes)
// 		assert.Empty(t, status.digicert)
// 		assert.Empty(t, status.customSCEPProxy)

// 		mt.invalid = &fleet.InvalidArgumentError{}
// 		mt.newAppConfig = &fleet.AppConfig{}
// 		mt.newAppConfig.Integrations.DigiCert.Set = true
// 		mt.newAppConfig.Integrations.DigiCert.Valid = true
// 		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "digicert", ErrMissingLicense.Error())

// 		mt.invalid = &fleet.InvalidArgumentError{}
// 		mt.newAppConfig = &fleet.AppConfig{}
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Set = true
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Valid = true
// 		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "custom_scep_proxy", ErrMissingLicense.Error())
// 	})

// 	t.Run("digicert keep old value", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.ctx = license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
// 		mt.oldAppConfig = mt.newAppConfig
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		mt.newAppConfig = &fleet.AppConfig{}
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.ndes)
// 		assert.Empty(t, status.digicert)
// 		assert.Empty(t, status.customSCEPProxy)
// 		assert.Len(t, mt.appConfig.Integrations.DigiCert.Value, 1)
// 	})

// 	t.Run("custom_scep keep old value", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.ctx = license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
// 		mt.oldAppConfig = mt.newAppConfig
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		mt.newAppConfig = &fleet.AppConfig{}
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.ndes)
// 		assert.Empty(t, status.digicert)
// 		assert.Empty(t, status.customSCEPProxy)
// 		assert.Len(t, mt.appConfig.Integrations.CustomSCEPProxy.Value, 1)
// 	})

// 	t.Run("missing server private key", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.svc.config.Server.PrivateKey = ""
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert", "private key")

// 		mt = setUpCustomSCEP()
// 		mt.svc.config.Server.PrivateKey = ""
// 		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy", "private key")
// 	})

// 	t.Run("invalid integration name", func(t *testing.T) {
// 		testCases := []struct {
// 			testName      string
// 			name          string
// 			errorContains []string
// 		}{
// 			{
// 				testName:      "empty",
// 				name:          "",
// 				errorContains: []string{"CA name cannot be empty"},
// 			},
// 			{
// 				testName:      "NDES",
// 				name:          "NDES",
// 				errorContains: []string{"CA name cannot be NDES"},
// 			},
// 			{
// 				testName:      "too long",
// 				name:          strings.Repeat("a", 256),
// 				errorContains: []string{"CA name cannot be longer than"},
// 			},
// 			{
// 				testName:      "invalid characters",
// 				name:          "a/b",
// 				errorContains: []string{"Only letters, numbers and underscores allowed"},
// 			},
// 		}

// 		for _, tc := range testCases {
// 			t.Run(tc.testName, func(t *testing.T) {
// 				baseErrorContains := tc.errorContains
// 				mt := setUpDigiCert()
// 				mt.newAppConfig = getAppConfigWithDigiCertIntegration(mockDigiCertServer.URL, tc.name)
// 				status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 				require.NoError(t, err)
// 				errorContains := baseErrorContains
// 				errorContains = append(errorContains, "integrations.digicert.name")
// 				checkExpectedCAValidationError(t, mt.invalid, status, errorContains...)

// 				mt = setUpCustomSCEP()
// 				mt.newAppConfig = getAppConfigWithSCEPIntegration("https://example.com", tc.name)
// 				status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 				require.NoError(t, err)
// 				errorContains = baseErrorContains
// 				errorContains = append(errorContains, "integrations.custom_scep_proxy.name")
// 				checkExpectedCAValidationError(t, mt.invalid, status, errorContains...)
// 			})
// 		}
// 	})

// 	t.Run("invalid digicert URL", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].URL = ""
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.url",
// 			"empty url")

// 		mt = setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].URL = "nonhttp://bad.com"
// 		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.url",
// 			"URL must be https or http")
// 	})

// 	t.Run("invalid custom_scep URL", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].URL = ""
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.url",
// 			"empty url")

// 		mt = setUpCustomSCEP()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].URL = "nonhttp://bad.com"
// 		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.url",
// 			"URL must be https or http")
// 	})

// 	t.Run("duplicate digicert integration name", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value = append(mt.newAppConfig.Integrations.DigiCert.Value,
// 			mt.newAppConfig.Integrations.DigiCert.Value[0])
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.name",
// 			"name is already used by another certificate authority")
// 	})

// 	t.Run("duplicate custom_scep integration name", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value = append(mt.newAppConfig.Integrations.CustomSCEPProxy.Value,
// 			mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0])
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.name",
// 			"name is already used by another certificate authority")
// 	})

// 	t.Run("same digicert and custom_scep integration name", func(t *testing.T) {
// 		mtSCEP := setUpCustomSCEP()
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy = mtSCEP.newAppConfig.Integrations.CustomSCEPProxy
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Name = mt.newAppConfig.Integrations.DigiCert.Value[0].Name
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.name",
// 			"name is already used by another certificate authority")
// 	})

// 	t.Run("digicert more than 1 user principal name", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames = append(mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames,
// 			"another")
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_user_principal_names",
// 			"one certificate user principal name")
// 	})

// 	t.Run("digicert empty user principal name", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames = []string{" "}
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_user_principal_names",
// 			"user principal name cannot be empty")
// 	})

// 	t.Run("digicert Fleet vars in user principal name", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames[0] = "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + " ${FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "}"
// 		_, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)

// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames[0] = "$FLEET_VAR_BOZO"
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_user_principal_names",
// 			"FLEET_VAR_BOZO is not allowed")
// 	})

// 	t.Run("digicert Fleet vars in common name", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateCommonName = "${FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + "}${FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "}"
// 		_, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)

// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateCommonName = "$FLEET_VAR_BOZO"
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_common_name",
// 			"FLEET_VAR_BOZO is not allowed")
// 	})

// 	t.Run("digicert Fleet vars in seat id", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateSeatID = "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + " $FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial)
// 		_, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)

// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateSeatID = "$FLEET_VAR_BOZO"
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_seat_id",
// 			"FLEET_VAR_BOZO is not allowed")
// 	})

// 	t.Run("digicert API token not set", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].APIToken = fleet.MaskedPassword
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.api_token", "DigiCert API token must be set")
// 	})

// 	t.Run("custom_scep challenge not set", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Challenge = fleet.MaskedPassword
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.challenge", "Custom SCEP challenge must be set")
// 	})

// 	t.Run("digicert common name not set", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateCommonName = "\n\t"
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_common_name", "Common Name (CN) cannot be empty")
// 	})

// 	t.Run("digicert seat id not set", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateSeatID = "\t\n"
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_seat_id", "Seat ID cannot be empty")
// 	})

// 	t.Run("digicert happy path -- add one", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.customSCEPProxy)
// 		require.Len(t, status.digicert, 1)
// 		assert.Equal(t, caStatusAdded, status.digicert[mt.newAppConfig.Integrations.DigiCert.Value[0].Name])
// 		require.Len(t, mt.appConfig.Integrations.DigiCert.Value, 1)
// 		assert.True(t, mt.newAppConfig.Integrations.DigiCert.Value[0].Equals(&mt.appConfig.Integrations.DigiCert.Value[0]))
// 	})

// 	t.Run("digicert happy path -- delete one", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.oldAppConfig = mt.newAppConfig
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		mt.newAppConfig = &fleet.AppConfig{
// 			Integrations: fleet.Integrations{
// 				DigiCert: optjson.Slice[fleet.DigiCertCA]{
// 					Set:   true,
// 					Valid: true,
// 				},
// 			},
// 		}
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.customSCEPProxy)
// 		require.Len(t, status.digicert, 1)
// 		assert.Equal(t, caStatusDeleted, status.digicert[mt.oldAppConfig.Integrations.DigiCert.Value[0].Name])
// 		assert.False(t, mt.appConfig.Integrations.DigiCert.Valid)
// 	})

// 	t.Run("digicert API token not set on modify", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.oldAppConfig.Integrations.DigiCert.Value = append(mt.oldAppConfig.Integrations.DigiCert.Value,
// 			mt.newAppConfig.Integrations.DigiCert.Value[0])
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].URL = "https://new.com"
// 		mt.newAppConfig.Integrations.DigiCert.Value[0].APIToken = ""
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.api_token", "DigiCert API token must be set when modifying")
// 	})

// 	t.Run("digicert happy path -- add one, delete one, modify one", func(t *testing.T) {
// 		mt := setUpDigiCert()
// 		mt.newAppConfig.Integrations.DigiCert = optjson.Slice[fleet.DigiCertCA]{
// 			Set:   true,
// 			Valid: true,
// 			Value: []fleet.DigiCertCA{
// 				{
// 					Name:                          "add",
// 					URL:                           mockDigiCertServer.URL,
// 					APIToken:                      "api_token",
// 					ProfileID:                     "profile_id",
// 					CertificateCommonName:         "common_name",
// 					CertificateUserPrincipalNames: []string{"user_principal_name"},
// 					CertificateSeatID:             "seat_id",
// 				},
// 				{
// 					Name:                          "modify",
// 					URL:                           mockDigiCertServer.URL,
// 					APIToken:                      "api_token",
// 					ProfileID:                     "profile_id",
// 					CertificateCommonName:         "common_name",
// 					CertificateUserPrincipalNames: nil,
// 					CertificateSeatID:             "seat_id",
// 				},
// 				{
// 					Name:                          "same",
// 					URL:                           mockDigiCertServer.URL,
// 					APIToken:                      "api_token",
// 					ProfileID:                     "profile_id",
// 					CertificateCommonName:         "other_cn",
// 					CertificateUserPrincipalNames: nil,
// 					CertificateSeatID:             "seat_id",
// 				},
// 			},
// 		}
// 		mt.oldAppConfig.Integrations.DigiCert = optjson.Slice[fleet.DigiCertCA]{
// 			Set:   true,
// 			Valid: true,
// 			Value: []fleet.DigiCertCA{
// 				{
// 					Name:                          "delete",
// 					URL:                           mockDigiCertServer.URL,
// 					APIToken:                      "api_token",
// 					ProfileID:                     "profile_id",
// 					CertificateCommonName:         "common_name",
// 					CertificateUserPrincipalNames: []string{"user_principal_name"},
// 					CertificateSeatID:             "seat_id",
// 				},
// 				{
// 					Name:                          "modify",
// 					URL:                           mockDigiCertServer.URL,
// 					APIToken:                      "api_token",
// 					ProfileID:                     "profile_id",
// 					CertificateCommonName:         "common_name",
// 					CertificateUserPrincipalNames: []string{"user_principal_name"},
// 					CertificateSeatID:             "seat_id",
// 				},
// 				{
// 					Name:                          "same",
// 					URL:                           mockDigiCertServer.URL,
// 					APIToken:                      "api_token",
// 					ProfileID:                     "profile_id",
// 					CertificateCommonName:         "other_cn",
// 					CertificateUserPrincipalNames: nil,
// 					CertificateSeatID:             "seat_id",
// 				},
// 			},
// 		}
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.customSCEPProxy)
// 		require.Len(t, status.digicert, 3)
// 		assert.Equal(t, caStatusAdded, status.digicert["add"])
// 		assert.Equal(t, caStatusEdited, status.digicert["modify"])
// 		assert.Equal(t, caStatusDeleted, status.digicert["delete"])
// 		require.Len(t, mt.appConfig.Integrations.DigiCert.Value, 3)
// 	})

// 	t.Run("custom_scep happy path -- add one", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.digicert)
// 		require.Len(t, status.customSCEPProxy, 1)
// 		assert.Equal(t, caStatusAdded, status.customSCEPProxy[mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Name])
// 		require.Len(t, mt.appConfig.Integrations.CustomSCEPProxy.Value, 1)
// 		assert.True(t, mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Equals(&mt.appConfig.Integrations.CustomSCEPProxy.Value[0]))
// 	})

// 	t.Run("custom_scep happy path -- delete one", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.oldAppConfig = mt.newAppConfig
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		mt.newAppConfig = &fleet.AppConfig{
// 			Integrations: fleet.Integrations{
// 				CustomSCEPProxy: optjson.Slice[fleet.CustomSCEPProxyCA]{
// 					Set:   true,
// 					Valid: true,
// 				},
// 			},
// 		}
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.digicert)
// 		require.Len(t, status.customSCEPProxy, 1)
// 		assert.Equal(t, caStatusDeleted, status.customSCEPProxy[mt.oldAppConfig.Integrations.CustomSCEPProxy.Value[0].Name])
// 		assert.False(t, mt.appConfig.Integrations.CustomSCEPProxy.Valid)
// 	})

// 	t.Run("custom_scep API token not set on modify", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.oldAppConfig.Integrations.CustomSCEPProxy.Value = append(mt.oldAppConfig.Integrations.CustomSCEPProxy.Value,
// 			mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0])
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].URL = "https://new.com"
// 		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Challenge = ""
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.challenge",
// 			"Custom SCEP challenge must be set when modifying")
// 	})

// 	t.Run("custom_scep happy path -- add one, delete one, modify one", func(t *testing.T) {
// 		mt := setUpCustomSCEP()
// 		mt.newAppConfig.Integrations.CustomSCEPProxy = optjson.Slice[fleet.CustomSCEPProxyCA]{
// 			Set:   true,
// 			Valid: true,
// 			Value: []fleet.CustomSCEPProxyCA{
// 				{
// 					Name:      "add",
// 					URL:       "https://example.com",
// 					Challenge: "challenge",
// 				},
// 				{
// 					Name:      "modify",
// 					URL:       "https://example.com",
// 					Challenge: "challenge",
// 				},
// 				{
// 					Name:      "SCEP_WIFI", // same
// 					URL:       "https://example.com",
// 					Challenge: "challenge",
// 				},
// 			},
// 		}
// 		mt.oldAppConfig.Integrations.CustomSCEPProxy = optjson.Slice[fleet.CustomSCEPProxyCA]{
// 			Set:   true,
// 			Valid: true,
// 			Value: []fleet.CustomSCEPProxyCA{
// 				{
// 					Name:      "delete",
// 					URL:       "https://example.com",
// 					Challenge: "challenge",
// 				},
// 				{
// 					Name:      "modify",
// 					URL:       "https://modify.com",
// 					Challenge: "challenge",
// 				},
// 				{
// 					Name:      "SCEP_WIFI", // same
// 					URL:       "https://example.com",
// 					Challenge: fleet.MaskedPassword,
// 				},
// 			},
// 		}
// 		mt.appConfig = mt.oldAppConfig.Copy()
// 		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
// 		require.NoError(t, err)
// 		assert.Empty(t, mt.invalid.Errors)
// 		assert.Empty(t, status.digicert)
// 		require.Len(t, status.customSCEPProxy, 3)
// 		assert.Equal(t, caStatusAdded, status.customSCEPProxy["add"])
// 		assert.Equal(t, caStatusEdited, status.customSCEPProxy["modify"])
// 		assert.Equal(t, caStatusDeleted, status.customSCEPProxy["delete"])
// 		require.Len(t, mt.appConfig.Integrations.CustomSCEPProxy.Value, 3)
// 	})
// }
