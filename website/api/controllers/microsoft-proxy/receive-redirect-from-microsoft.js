module.exports = {


  friendlyName: 'Get tenants admin consent status',


  description: 'Updates the admin consent status of a MicrosoftComplianceTenant record and if the admin consented, completes the setup of the new tenant.',


  inputs: {
    tenant: {
      type: 'string',
      description: 'The entra tenant ID of a Microsoft Compliance Partner tenant'
    },
    state: {
      type: 'string',
      description: 'A token used to authenticate this request for a provided entra tenant.'
    },
    error: {
      type: 'string',
      description: 'This value is only present if an entra admin did approve the permissions for the compliance partner policy'
    },
    error_description: {// eslint-disable-line camelcase
      type: 'string',
      description: 'This value is only present if an entra admin did approve the permissions for the compliance partner policy'
    }
  },


  exits: {
    success: { description: 'The admin consent status of compliance tenant was successfully updated.', responseType: 'redirect'},
    redirect: { responseType: 'redirect' },
    adminDidNotConsent: { description: 'An entra admin did not grant permissions to the Fleet compliance partner application for entra', responseType: 'ok'},
    noMatchingTenant: { description: 'This request did not match any existing Microsoft compliance tenant', responseType: 'notFound'},
    badRequest: { description: 'This request is missing a tenant or state value', responseType: 'badRequest'}
  },


  fn: async function ({tenant, state, error, error_description}) {// eslint-disable-line camelcase

    // If an error or error_description are provided, then the admin did not consent, and we will return a 200 response.
    if(error || error_description){// eslint-disable-line camelcase
      throw 'adminDidNotConsent';
    } else if (!tenant || !state) {
      throw 'badRequest';
    }

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: tenant, stateTokenForAdminConsent: state});
    // If no tenant was found that matches the provided tenant id and state, return a not found response.
    if(!informationAboutThisTenant) {
      throw 'noMatchingTenant';
    } else {
      let fleetInstanceUrlToRedirectTo = informationAboutThisTenant.fleetInstanceUrl + '/settings/integrations/conditional-access';

      // Use the microsoftProcy.getAccessTokenAndApiUrls helper to get an access token and API urls for this tenant.
      let accessTokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
        complianceTenantRecordId: informationAboutThisTenant.id
      });

      let manageApiAccessToken = accessTokenAndApiUrls.manageApiAccessToken;
      let graphAccessToken = accessTokenAndApiUrls.graphAccessToken;
      let tenantDataSyncUrl = accessTokenAndApiUrls.tenantDataSyncUrl;
      // let partnerCompliancePoliciesUrl = accessTokenAndApiUrls.partnerCompliancePoliciesUrl;

      // Send a request to provision the new compliance tenant.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'PUT',
        url: `${tenantDataSyncUrl}/PartnerTenants(guid'${informationAboutThisTenant.entraTenantId}')}?api-version=1.6`,
        headers: {
          'Authorization': `Bearer ${manageApiAccessToken}`
        },
        body: {
          Provisioned: 1,// 1 = provisioned, 2 = deprovisioned.
          PartnerEnrollmentUrl: `https://fleetdm.com/microsoft-compliance-partner/enroll`,
          PartnerRemediationUrl: `https://fleetdm.com/microsoft-compliance-partner/remediate`,
        }
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`an error occurred when provisioning a new Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return {redirect: fleetInstanceUrlToRedirectTo };
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




      // (2025-05-14) Testing note: If a request after this request fails, the request to create a policy will return a 409 error on subsequent runs.
      // Now send a request to create a new compliance policy on the tenant.
      let createPolicyResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'POST',
        url: `${tenantDataSyncUrl}/PartnerCompliancePolicies?api-version=1.6`,
        headers: {
          'Authorization': `Bearer ${manageApiAccessToken}`,
          'Content-Type': 'application/json'
        },
        body: {
          DisplayName: 'Fleet compliance policy',
          Description: 'Compliance policy managed by Fleet',
          Platform: 'macOS',
          PartnerManagedCompliance: true
        }
      }).tolerate({raw:{statusCode: 409}}, async ()=>{
        // If a partner compliance policy already exists, send a request to get all policies and return the previously created policy.
        let getPoliciesResponse = await sails.helpers.http.sendHttpRequest.with({
          method: 'GET',
          url: `${tenantDataSyncUrl}/PartnerCompliancePolicies?api-version=1.6`,
          headers: {
            'Authorization': `Bearer ${manageApiAccessToken}`,
            'Content-Type': 'application/json'
          }
        });
        return getPoliciesResponse;
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`An error occurred when creating a new compliance policy on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return {redirect: fleetInstanceUrlToRedirectTo };
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


      let parsedPoliciesResponse;
      try {
        parsedPoliciesResponse = JSON.parse(createPolicyResponse.body);
      } catch(err){
        sails.log.warn(`An error occured when parsing the JSON response body from the PartnerCompliancePolicies endpoint for a microsoft compliance tenant. full error`, err);
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        return {redirect: fleetInstanceUrlToRedirectTo };
      }
      let createdPolicyId = parsedPoliciesResponse.value[0].Id;

      // Use the Microsoft Graph API to retreive the ID of the default "All users" group to assign the policy to.
      let groupResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'GET',
        url: `https://graph.microsoft.com/v1.0/groups`,
        headers: {
          'Authorization': `Bearer ${graphAccessToken}`
        }
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`An error occurred when sending a request to find the default "All users" group on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return {redirect: fleetInstanceUrlToRedirectTo };
      });
      // Get the ID returned in the response.
      let parsedGroupResponse;
      try {
        parsedGroupResponse = JSON.parse(groupResponse.body);
      } catch(err){
        sails.log.warn(`An error occured when parsing the JSON response body returned by the Microsoft graph API for a new Microsoft compliance tenant. full error`, err);
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        return {redirect: fleetInstanceUrlToRedirectTo };
      }
      let groupId = parsedGroupResponse.value[0].id;


      // Send a request to assign the new compliance policy to the "All users" group.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'POST',
        url: `${tenantDataSyncUrl}/PartnerCompliancePolicies(guid'${encodeURIComponent(createdPolicyId)}')/Assign?api-version=1.6`,
        headers: {
          'Authorization': `Bearer ${manageApiAccessToken}`,
          'Content-Type': 'application/json'
        },
        body: {
          assignments: [
            groupId
          ]
        }
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`An error occurred when sending a assign a new compliance policy to "All users" on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return {redirect: fleetInstanceUrlToRedirectTo };
      });
      // Example response:
      // {
      //   “odata.metadata”: “…”,
      //   "value": “{\”assignments\":[\"GuidValue\",\"GuidValue\"]}”
      // }

      // Update the database record to show that setup was completed for this compliance tenant.
      await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({
        setupCompleted: true,
        adminConsented: true,
        stateTokenForAdminConsent: undefined,
      });

      // Redirect the user to their Fleet instance.
      return fleetInstanceUrlToRedirectTo;
    }

  }


};
