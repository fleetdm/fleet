module.exports = {


  friendlyName: 'Setup compliance partner tenant',


  description: '',


  inputs: {

  },


  exits: {
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
  },


  fn: async function () {

    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    if(!this.req.headers['Authorization']) {
      return this.res.unauthorized();
    }

    let authHeaderValue = this.req.headers['Authorization']
    let tokenForThisRequest = authHeaderValue.split('Bearer ')[1];
    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({apiKey: tokenForThisRequest});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided API key.'})// TODO: return a more clear error.
    }

    let accessTokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = accessTokenAndApiUrls.accessToken;
    let tenantDataSyncUrl = accessTokenAndApiUrls.tenantDataSyncUrl;
    let partnerCompliancePoliciesUrl = accessTokenAndApiUrls.partnerCompliancePoliciesUrl;

    // Provision the new tenant.
    await sails.helpers.http.sendHtttpRequest.with({
      method: 'PUT',
      url: `${tenantDataSyncUrl}/PartnerTenants(guid${encodeURIComponent(informationAboutThisTenant.entraTenantId)})}?api-version=1.0`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
      body: {
        Provisioned: 1,// 1 = provisioned, 2 = deprovisioned.
        PartnerEnrollmentUrl: '', //TODO: how do we get this, the example in microsoft's docs are using customer.com/enrollment, so does this need to be a value of a url on the connected Fleet instance?
        PartnerRemediationUrl: '', // TODO: same as the above.
      }
    });
    // Example response:
    // HTTP/1.1 200 OK
    // {
    //   "Key": "bd0d0eec-06a2-4a0f-9296-184e76f67e5a",
    //   "ContextId": "<Entra tenant id>", // This is the Entra ID tenant ID
    //   "Provisioned": 1,
    //   "HttpStatusCode": 200,
    //   "ErrorDetail": null
    // }

    // Now create a new Compliance policy.
    let createPolicyResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `${partnerCompliancePoliciesUrl}/PartnerCompliancePolicies?api-version=1.6`,
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json'
      },
      body: {
        DisplayName: 'Fleet compliance policy',
        Description: 'Compliance policy managed by Fleet',
        Platform: 'macOS',
        PartnerManagedCompliance: true
      }
    });

    // Example response:
    // HTTP/1.1 201 Created
    // {
    //   “odata.metadata”: “…”,
    //   “odata.id”: “…”,
    //   “odata.etag”: “…”,
    //   “Id”: “<GuidValue>”
    //   "DisplayName": "Partner compliance policy",
    //   “Description”: “Policy description”,
    //   “Platform”: “iOS”,
    //   “PartnerPolicyId”: “<GuidValue>”,
    //   “PartnerManagedCompliance”: true
    // }


    // Get the id of the new policy from the API response.
    let createdPolicyId = createPolicyResponse.Id;

    // Find a user group to assign the policy to.
    // TODO: this currently looks for a user group named "All users"
    let groupResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'GET',
      url: `https://graph.microsoft.com/v1.0/groups?$filter=${encodeURIComponent('displayName eq 'All Users' and securityEnabled eq true')}`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    let groupId = groupResponse.value[0].id;

    await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `${partnerCompliancePoliciesUrl}/PartnerCompliancePolicies(guid'${encodeURIComponent(createdPolicyId)}')/Assign?api-version=1.6`,
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json'
      },
      body: {
        assignments: [
          groupId
        ]
      }
    });
    // Example response:
    // {
    //   “odata.metadata”: “…”,
    //   "value": “{\”assignments\":[\"GuidValue\",\"GuidValue\"]}”
    // }


    await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({
      setupCompleted: true,
    });

    // return a 200 response if everything was setup correctly
    return;

  }


};
