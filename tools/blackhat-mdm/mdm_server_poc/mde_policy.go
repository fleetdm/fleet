package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// PolicyHandler is the HTTP handler assosiated with the enrollment protocol's policy endpoint.
func PolicyHandler(w http.ResponseWriter, r *http.Request) {
	// Read The HTTP Request body
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	body := string(bodyRaw)

	// Retrieve the MessageID From The Body For The Response
	messageID := strings.Replace(strings.Replace(regexp.MustCompile(`<a:MessageID>[\s\S]*?<\/a:MessageID>`).FindStringSubmatch(body)[0], "<a:MessageID>", "", -1), "</a:MessageID>", "", -1)

	response := []byte(`
			<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
			  <s:Header>
				<Action mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPoliciesResponse</Action>
				<a:RelatesTo>` + messageID + `</a:RelatesTo>
			  </s:Header>
			  <s:Body xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
				<GetPoliciesResponse xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
				  <response>
					<policyID></policyID>
					<policyFriendlyName xsi:nil="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"></policyFriendlyName>
					<nextUpdateHours xsi:nil="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"></nextUpdateHours>
					<policiesNotChanged xsi:nil="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"></policiesNotChanged>
					<policies>
					  <policy>
						<policyOIDReference>0</policyOIDReference>
						<cAs xsi:nil="true"></cAs>
						<attributes>
						  <commonName>Attributes</commonName>
						  <policySchema>2</policySchema>
						  <certificateValidity>
							<validityPeriodSeconds>1209600</validityPeriodSeconds>
							<renewalPeriodSeconds>172800</renewalPeriodSeconds>
						  </certificateValidity>
						  <permission>
							<enroll>true</enroll>
							<autoEnroll>false</autoEnroll>
						  </permission>
						  <privateKeyAttributes>
							<minimalKeyLength>2048</minimalKeyLength>
							<keySpec xsi:nil="true"></keySpec>
							<keyUsageProperty xsi:nil="true"></keyUsageProperty>
							<permissions xsi:nil="true"></permissions>
							<algorithmOIDReference xsi:nil="true"></algorithmOIDReference>
							<cryptoProviders xsi:nil="true"></cryptoProviders>
						  </privateKeyAttributes>
						  <revision>
							<majorRevision>101</majorRevision>
							<minorRevision>0</minorRevision>
						  </revision>
						  <supersededPolicies xsi:nil="true"></supersededPolicies>
						  <privateKeyFlags xsi:nil="true"></privateKeyFlags>
						  <subjectNameFlags xsi:nil="true"></subjectNameFlags>
						  <enrollmentFlags xsi:nil="true"></enrollmentFlags>
						  <generalFlags xsi:nil="true"></generalFlags>
						  <hashAlgorithmOIDReference>0</hashAlgorithmOIDReference>
						  <rARequirements xsi:nil="true"></rARequirements>
						  <keyArchivalAttributes xsi:nil="true"></keyArchivalAttributes>
						  <extensions xsi:nil="true"></extensions>
						</attributes>
					  </policy>
					</policies>
				  </response>
				  <oIDs>
					<oID>
					  <value>1.3.14.3.2.29</value>
					  <group>1</group>
					  <oIDReferenceID>0</oIDReferenceID>
					  <defaultName>szOID_NIST_sha256</defaultName>
					</oID>
				  </oIDs>
				</GetPoliciesResponse>
			  </s:Body>
			</s:Envelope>`)

	// Return response body
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Write(response)
}
