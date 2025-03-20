module.exports = {


  friendlyName: 'Setup compliance partner tenant',


  description: 'Sets up a new Microsoft compliance integration in a users intune instance.',


  inputs: {
    entraTenantId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      requried: true,
    },
  },


  exits: {
    success: {description: 'A Microsoft compliance tenant was successfully provisioned or has already been sucessfully provisioned.'}
  },


  fn: async function ({entraTenantId, fleetServerSecret}) {


    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId});
    if(!informationAboutThisTenant) {
      return new Error({error: 'Invalid Tenant ID: No MicrosoftComplianceTenant record was found that matches the provided API key.'});// TODO: return a more clear error.
    }
    if(informationAboutThisTenant.fleetServerSecret !== fleetServerSecret){
      return new Error({error: 'Invalid secret: The provided fleetServerSecret does not match the secret for the provided tenant ID.'});
    }
    // If setup was already completed for this tenant, return a 200 status code.
    if(informationAboutThisTenant.setupCompleted){
      return;
    }

    // Use the microsoftProcy.getAccessTokenAndApiUrls helper to get an access token and API urls for this tenant.
    let accessTokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = accessTokenAndApiUrls.accessToken;
    let tenantDataSyncUrl = accessTokenAndApiUrls.tenantDataSyncUrl;
    let partnerCompliancePoliciesUrl = accessTokenAndApiUrls.partnerCompliancePoliciesUrl;

    // Send a request to provision the new compliance tenant.
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
    }).intercept((err)=>{
      return new Error({error: `an error occurred when provisioning a new Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
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

    // Now send a request to create a new compliance policy on the tenant.
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
    }).intercept((err)=>{
      return new Error({error: `An error occurred when creating a new compliance policy on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`})
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

    // Use the Microsoft Graph API to retreive the ID of the default "All users" group to assign the policy to.
    let groupResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'GET',
      url: `https://graph.microsoft.com/v1.0/groups?$filter=${encodeURIComponent(`displayName eq 'All Users' and securityEnabled eq true`)}`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    }).intercept((err)=>{
      return new Error({error: `An error occurred when sending a request to find the default "All users" group on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`})
    });
    // Get the ID returned in the response.
    let groupId = groupResponse.value[0].id;


    // Send a request to assign the new compliance policy to the "All users" group.
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
    }).intercept((err)=>{
      return new Error({error: `An error occurred when sending a assign a new compliance policy to "All users" on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`})
    });
    // Example response:
    // {
    //   “odata.metadata”: “…”,
    //   "value": “{\”assignments\":[\"GuidValue\",\"GuidValue\"]}”
    // }

    // Update the databse record to show that setup was completed for this compliance tenant.
    await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({
      setupCompleted: true,
    });

    // return a 200 response if everything was setup correctly
    return;

  }


};
