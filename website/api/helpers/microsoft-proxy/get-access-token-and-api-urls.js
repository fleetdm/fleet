module.exports = {


  friendlyName: 'Get access token and api urls',


  description: '',


  inputs: {
    complianceTenantRecordId: {
      type: 'number',
      required: true,
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Access token and api urls',
    },

  },


  fn: async function ({complianceTenantRecordId}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({id: complianceTenantRecordId});
    if(!informationAboutThisTenant) {
      return new Error(`No matching tenant record could be found with the specified ID. (${complianceTenantRecordId}`)
    }

    // Get an access token for this
    let accessTokenResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `https://login.microsoftonline.com/${informationAboutThisTenant.entraTenantId}/oauth2/v2.0/token`,
      enctype: 'application/x-www-form-urlencoded',
      body: {
        client_id: sails.config.custom.compliancePartnerClientId,
        scope: 'https://graph.microsoft.com/.default',
        client_secret: sails.config.custom.compliancePartnerClientSecret,
        grant_type: 'client_credentials'
      }
    });
    // TODO: return error if no access token.
    let accessToken = accessTokenResponse.body.access_token;
    // [?]: https://learn.microsoft.com/en-us/graph/api/resources/serviceprincipal
    let servicePrincipalResponse = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/servicePrincipals?$filter=${encodeURIComponent(`appId eq ${informationAboutThisTenant.entraTenantId}`)}`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    // TODO: verify that the servicePrincipal object is not nested in the response from the Microsoft graph API.
    let servicePrincipalObjectId = servicePrincipalResponse.id;

    // [?]: https://learn.microsoft.com/en-us/graph/api/group-list-endpoints
    let servicePrincipalEndpointResponse = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/servicePrincipals/${servicePrincipalObjectId}/endpoints`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
    });

    let endpointsInResponse = servicePrincipalEndpointResponse.value;
    let tenantDataSyncUrl = _.find(endpointsInResponse, {providerName: 'PartnerTenantDataSyncService'}).uri;
    let partnerCompliancePoliciesUrl = _.find(endpointsInResponse, {providerName: 'PartnerCompliancePolicies'}).uri;


    return {
      accessToken,
      tenantDataSyncUrl,
      partnerCompliancePoliciesUrl,
    };
  }


};

