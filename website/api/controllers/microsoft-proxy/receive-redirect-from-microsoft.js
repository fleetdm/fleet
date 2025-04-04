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
    success: { description: 'The admin consent status of compliance tenant was successfully updated.', responseType: 'ok'},
    adminDidNotConsent: { description: 'An entra admin did not grant permissions to the Fleet compliance partner application for entra', responseType: 'ok'}
  },


  fn: async function ({tenant, state, error, error_description}) {// eslint-disable-line camelcase

    // If an error or error_description are provided, then the admin did not consent, and we will return a 200 response.
    if(error || error_description){// eslint-disable-line camelcase
      throw 'adminDidNotConsent';
    }

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: tenant, stateTokenForAdminConsent: state});
    // If no tenant was found that matches the provided tenant id and state, return a not found response.
    if(!informationAboutThisTenant) {
      return this.res.notFound();
    } else {
      let fleetInstanceUrlToRedirectTo = informationAboutThisTenant.fleetInstanceUrl + '/settings/integrations/conditional-access';
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
          PartnerEnrollmentUrl: `${informationAboutThisTenant.fleetInstanceUrl}/enrollment`,
          PartnerRemediationUrl: `${informationAboutThisTenant.fleetInstanceUrl}/remediation`,
        }
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`an error occurred when provisioning a new Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return this.res.redirect(fleetInstanceUrlToRedirectTo);
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
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`An error occurred when creating a new compliance policy on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return this.res.redirect(fleetInstanceUrlToRedirectTo);
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
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`An error occurred when sending a request to find the default "All users" group on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return this.res.redirect(fleetInstanceUrlToRedirectTo);
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
      }).intercept(async (err)=>{
        await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({setupError:  `${require('util').inspect(err, {depth: null})}`});
        sails.log.warn(`An error occurred when sending a assign a new compliance policy to "All users" on a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return this.res.redirect(fleetInstanceUrlToRedirectTo);
      });
      // Example response:
      // {
      //   “odata.metadata”: “…”,
      //   "value": “{\”assignments\":[\"GuidValue\",\"GuidValue\"]}”
      // }

      // Update the databse record to show that setup was completed for this compliance tenant.
      await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({
        setupCompleted: true,
        adminConsented: true,
        stateTokenForAdminConsent: undefined,
      });

      // Redirect the user to their Fleet instance.
      return this.res.redirect(fleetInstanceUrlToRedirectTo);
    }

  }


};
