module.exports = {


  friendlyName: 'Get access token and api urls',


  description: 'Retreives an access token and the URLS of API endpoints for a Microsoft compliance tenant',


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
      return new Error(`No matching tenant record could be found with the specified ID. (${complianceTenantRecordId}`);
    }

    // Get an access token for this tenant
    let accessTokenResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `https://login.microsoftonline.com/${informationAboutThisTenant.entraTenantId}/oauth2/v2.0/token`,
      enctype: 'application/x-www-form-urlencoded',
      body: {
        client_id: sails.config.custom.compliancePartnerClientId,// eslint-disable-line camelcase
        scope: 'https://graph.microsoft.com/.default',
        client_secret: sails.config.custom.compliancePartnerClientSecret,// eslint-disable-line camelcase
        grant_type: 'client_credentials'// eslint-disable-line camelcase
      }
    });

    // Parse the json response body to get the access token.
    let accessToken;
    try {
      accessToken = JSON.parse(accessTokenResponse.body).access_token;
    } catch(err){
      throw new Error(`When sending a request to get an access token for a Microsoft compliance tenant, an error occured. full error: ${require('util').inspect(err)}`);
    }

    // (2025-04-01) TODO: test the code below once the test tenant has been configured to be able to use the Fleet compliance partner application.

    // [?]: https://learn.microsoft.com/en-us/graph/api/resources/serviceprincipal
    let servicePrincipalResponse = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/servicePrincipals?$filter=${encodeURIComponent(`appId eq '${sails.config.custom.compliancePartnerClientId}'`)}`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    let servicePrincipalObjectId = servicePrincipalResponse.value[0].id;

    // [?]: https://learn.microsoft.com/en-us/graph/api/group-list-endpoints
    let servicePrincipalEndpointResponse = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/servicePrincipals/${servicePrincipalObjectId}/endpoints`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
    });
    let endpointsInResponse = servicePrincipalEndpointResponse.value;
    // TODO: throw errors if these endpoints are missing.
    let tenantDataSyncUrl = _.find(endpointsInResponse, {providerName: 'PartnerTenantDataSyncService'}).uri;
    let partnerCompliancePoliciesUrl = _.find(endpointsInResponse, {providerName: 'PartnerCompliancePolicies'}).uri;

    return {
      accessToken,
      tenantDataSyncUrl,
      partnerCompliancePoliciesUrl,
    };
  }


};

