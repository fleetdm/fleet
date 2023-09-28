package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// DiscoveryHandler is the HTTP handler assosiated with the enrollment protocol's discovery endpoint.
func DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	// Return HTTP Status 200 Ok when a HTTP GET request is received.
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Read The HTTP Request body
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	body := string(bodyRaw)

	// Retrieve the MessageID From The Body For The Response
	messageID := strings.Replace(strings.Replace(regexp.MustCompile(`<a:MessageID>[\s\S]*?<\/a:MessageID>`).FindStringSubmatch(body)[0], "<a:MessageID>", "", -1), "</a:MessageID>", "", -1)

	var extraParams = ""
	if authPolicy == "Federated" {
		extraParams += "<AuthenticationServiceUrl>https://" + domain + "/EnrollmentServer/Auth</AuthenticationServiceUrl>"
	}

	// Create response payload
	response := []byte(`
		<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing">
		  <s:Header>
			<Action mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/DiscoverResponse</Action>
			<RelatesTo>` + messageID + `</RelatesTo>
			<ActivityId CorrelationId="735046d3-5b2c-4512-a7be-09e3da447abf" xmlns="http://schemas.microsoft.com/2004/09/ServiceModel/Diagnostics">735046d3-5b2c-4512-a7be-09e3da447abf</ActivityId>
		  </s:Header>
		  <s:Body>
			<DiscoverResponse xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
			  <DiscoverResult>
				<AuthPolicy>` + authPolicy + `</AuthPolicy>
				<EnrollmentVersion>4.0</EnrollmentVersion>
				<EnrollmentPolicyServiceUrl>https://` + domain + `/EnrollmentServer/Policy.svc</EnrollmentPolicyServiceUrl>
				<EnrollmentServiceUrl>https://` + domain + `/EnrollmentServer/Enrollment.svc</EnrollmentServiceUrl>
				<AuthenticationServiceUrl>https://` + domain + `/EnrollmentServer/Auth.svc</AuthenticationServiceUrl>				
			  </DiscoverResult>
			</DiscoverResponse>
		  </s:Body>
		</s:Envelope>`)

	// Return response body
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Write(response)
}
