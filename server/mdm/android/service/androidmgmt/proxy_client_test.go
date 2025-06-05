package androidmgmt

// import (
// 	"testing"
//
// 	"github.com/go-json-experiment/json"
// 	"github.com/stretchr/testify/require"
// 	"google.golang.org/api/androidmanagement/v1"
// )
//
// func TestProxy(t *testing.T) {
// 	type proxyEnterprise struct {
// 		FleetLicenseKey string `json:"fleetLicenseKey"`
// 		PubSubPushURL   string `json:"pubSubPushUrl"`
// 		EnterpriseToken string `json:"enterpriseToken"`
// 		SignupURLName   string `json:"signupUrlName"`
// 		Enterprise      androidmanagement.Enterprise
// 	}
//
// 	pe := proxyEnterprise{
// 		FleetLicenseKey: fleetLicenseKey,
// 		PubSubPushURL:   "https://example.com/push/endpoint/url",
// 		EnterpriseToken: "enterprise token",
// 		SignupURLName:   "signup url name",
// 		Enterprise: androidmanagement.Enterprise{
// 			EnabledNotificationTypes: []string{"type1"},
// 			Name:                     "enterprise name",
// 		},
// 	}
//
// 	reqBody, err := json.Marshal(pe)
// 	require.NoError(t, err)
// 	t.Log(string(reqBody))
//
// 	type proxyEnterpriseResponse struct {
// 		FleetServerSecret string `json:"fleetServerSecret"`
// 		Name              string `json:"name"`
// 	}
// 	var per proxyEnterpriseResponse
// 	require.NoError(t, json.Unmarshal(reqBody, &per))
// 	t.Logf("%+v", per)
//
// 	body := `{
//   "name": "enterprises/LC02zwvys4",
//   "fleetServerSecret": "4c18a6726a41423c1bb0df78c8257b"
// }`
// 	per = proxyEnterpriseResponse{}
// 	require.NoError(t, json.Unmarshal([]byte(body), &per))
// 	t.Logf("%+v", per)
//
// }
