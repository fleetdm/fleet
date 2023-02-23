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
				<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPoliciesResponse</a:Action>
				<a:RelatesTo>` + messageID + `</a:RelatesTo>
				</s:Header>
				<s:Body
					xmlns:xsd="http://www.w3.org/2001/XMLSchema"
					xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
					<GetPoliciesResponse xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
						<response>
							<policyID />
							<policyFriendlyName xsi:nil="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"/>
							<nextUpdateHours xsi:nil="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"/>
							<policiesNotChanged xsi:nil="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"/>
							<policies>
								<policy>
									<policyOIDReference>0</policyOIDReference>
									<cAs xsi:nil="true" />
									<attributes>
										<commonName>CEPUnitTest</commonName>
										<policySchema>3</policySchema>
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
											<keySpec xsi:nil="true" />
											<keyUsageProperty xsi:nil="true" />
											<permissions xsi:nil="true" />
											<algorithmOIDReference xsi:nil="true" />
											<cryptoProviders xsi:nil="true" />
										</privateKeyAttributes>
										<revision>
											<majorRevision>101</majorRevision>
											<minorRevision>0</minorRevision>
										</revision>
										<supersededPolicies xsi:nil="true" />
										<privateKeyFlags xsi:nil="true" />
										<subjectNameFlags xsi:nil="true" />
										<enrollmentFlags xsi:nil="true" />
										<generalFlags xsi:nil="true" />
										<hashAlgorithmOIDReference>0</hashAlgorithmOIDReference>
										<rARequirements xsi:nil="true" />
										<keyArchivalAttributes xsi:nil="true" />
										<extensions xsi:nil="true" />
									</attributes>
								</policy>
							</policies>
						</response>
						<oIDs>
							<oID>
								<value>1.3.14.3.2.29</value>
								<group>1</group>
								<oIDReferenceID>0</oIDReferenceID>
								<defaultName> szOID_NIST_sha256</defaultName>
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
