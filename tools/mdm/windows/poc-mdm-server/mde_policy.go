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
			<s:Envelope
	xmlns:s="http://www.w3.org/2003/05/soap-envelope"
	xmlns:a="http://www.w3.org/2005/08/addressing">
	<s:Header>
		<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPoliciesResponse</a:Action>
		<ActivityId CorrelationId="327cf31e-dade-4652-8e24-986f12d3464d"
			xmlns="http://schemas.microsoft.com/2004/09/ServiceModel/Diagnostics">a4d836c0-1a77-4351-ba85-40a3df130178
		</ActivityId>
		<a:RelatesTo>` + messageID + `</a:RelatesTo>
	</s:Header>
	<s:Body>
		<GetPoliciesResponse
			xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
			<response>
				<policyFriendlyName p6:nil="true"
					xmlns:p6="http://www.w3.org/2001/XMLSchema-instance" />
					<nextUpdateHours p6:nil="true"
						xmlns:p6="http://www.w3.org/2001/XMLSchema-instance" />
						<policiesNotChanged p6:nil="true"
							xmlns:p6="http://www.w3.org/2001/XMLSchema-instance" />
							<policies>
								<policy>
									<policyOIDReference>0</policyOIDReference>
									<cAs p8:nil="true"
										xmlns:p8="http://www.w3.org/2001/XMLSchema-instance" />
										<attributes>
											<policySchema>3</policySchema>
											<privateKeyAttributes>
												<minimalKeyLength>2048</minimalKeyLength>
												<keySpec p10:nil="true"
													xmlns:p10="http://www.w3.org/2001/XMLSchema-instance" />
													<keyUsageProperty p10:nil="true"
														xmlns:p10="http://www.w3.org/2001/XMLSchema-instance" />
														<permissions p10:nil="true"
															xmlns:p10="http://www.w3.org/2001/XMLSchema-instance" />
															<algorithmOIDReference>1</algorithmOIDReference>
															<cryptoProviders>
																<provider>Microsoft Platform Crypto Provider</provider>
																<provider>Microsoft Software Key Storage Provider</provider>
															</cryptoProviders>
														</privateKeyAttributes>
														<supersededPolicies p9:nil="true"
															xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
															<privateKeyFlags p9:nil="true"
																xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																<subjectNameFlags p9:nil="true"
																	xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																	<enrollmentFlags p9:nil="true"
																		xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																		<generalFlags p9:nil="true"
																			xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																			<hashAlgorithmOIDReference>0</hashAlgorithmOIDReference>
																			<rARequirements p9:nil="true"
																				xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																				<keyArchivalAttributes p9:nil="true"
																					xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																					<extensions p9:nil="true"
																						xmlns:p9="http://www.w3.org/2001/XMLSchema-instance" />
																						<attestation>
																							<attestationFailureBehavior>RetryOnError</attestationFailureBehavior>
																							<operationTimeout>30</operationTimeout>
																						</attestation>
																					</attributes>
																				</policy>
																			</policies>
																		</response>
																		<cAs />
																		<oIDs>
																			<oID>
																				<value>2.16.840.1.101.3.4.2.1</value>
																				<group>4</group>
																				<oIDReferenceID>0</oIDReferenceID>
																				<defaultName>szOID_NIST_sha256</defaultName>
																			</oID>
																			<oID>
																				<value>1.2.840.113549.1.1.1</value>
																				<group>3</group>
																				<oIDReferenceID>1</oIDReferenceID>
																				<defaultName>szOID_RSA_RSA</defaultName>
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
