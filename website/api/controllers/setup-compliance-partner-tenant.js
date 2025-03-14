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

    let authHeaderValue = this.req.headers['Authorization'];
    let tokenForThisRequest = authHeaderValue.split('Bearer ')[1];
    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({apiKey: tokenForThisRequest});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided API key.'});// TODO: return a more clear error.
    }
    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = tokenAndApiUrls.accessToken;
    let tenantDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;
    let partnerCompliancePoliciesUrl = tokenAndApiUrls.partnerCompliancePoliciesUrl;

    // Provision the new tenant.

    await sails.helpers.http.sendHtttpRequest.with({
      method: 'PUT',
      url: `${tenantDataSyncUrl}/${encodeURIComponent(`PartnerTenants(guid${informationAboutThisTenant.entraTenantId}`)}?api-version=1.0`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
      body: {
        Provisioned: 1,// 1 = provisioned, 2 = deprovisioned.
        PartnerEnrollmentUrl: '', //TODO: how do we get this, the example in microsoft's docs are using customer.com/enrollment, so does this need to be a value of a url on the connected Fleet instance?
        PartnerRemediationUrl: '', // TODO: same as the above.
      }
    });


    // Now create a new policy for this tenant

    // Create Compliance Policy
    let createPolicyResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `${partnerCompliancePoliciesUrl}/PartnerCompliancePolicies?api-version=1.6`,
      headers: {
        'Authorization': `Bearer ${accessToken}`,
      },
      body: {
        DisplayName: 'Fleet compliance policy',
        Description: 'Compliance policy managed by Fleet',
        Platform: 'macOS',
        PartnerManagedCompliance: true
      }
    });

    let createdPolicyId = createPolicyResponse.Id;

    // Find a group to assign the policy to.
    let groupResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'GET',
      // TODO: this criteria assumes that there is a user group named "All Users" on the tenant's entra instance.
      url: `https://graph.microsoft.com/v1.0/groups?$filter=${encodeURIComponent("displayName eq 'All Users' and securityEnabled eq true")}`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    let allUsersGroupId = groupResponse.body.value[0].id;

    // Assign the policy we creaed to the group on the entra instance.
    await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `${partnerCompliancePoliciesUrl}/PartnerCompliancePolicies(guid'${createdPolicyId}')/Assign?api-version=1.6`,
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json'
      },
      body: {
        assignments: [
          allUsersGroupId
        ]
      }
    });


    // update the Database record for this tenant.
    await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({
      setupCompleted: true,
      macosCompliancePolicyGuid: createdPolicyId,
    });

    // return a 200 response if everything was setup correctly
    return;

  }


};
