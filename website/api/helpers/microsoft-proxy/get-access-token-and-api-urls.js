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

    // Get a graph access token for this tenant
    let graphAccessTokenResponse = await sails.helpers.http.sendHttpRequest.with({
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
    // Get a management API access token for this tenant
    let manageAccessTokenResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `https://login.microsoftonline.com/${informationAboutThisTenant.entraTenantId}/oauth2/v2.0/token`,
      enctype: 'application/x-www-form-urlencoded',
      body: {
        client_id: sails.config.custom.compliancePartnerClientId,// eslint-disable-line camelcase
        scope: 'https://api.manage.microsoft.com//.default',
        client_secret: sails.config.custom.compliancePartnerClientSecret,// eslint-disable-line camelcase
        grant_type: 'client_credentials'// eslint-disable-line camelcase
      }
    });

    // Parse the json response body to get the access token.
    let graphAccessToken;
    let manageApiAccessToken;
    try {
      graphAccessToken = JSON.parse(graphAccessTokenResponse.body).access_token;
      manageApiAccessToken = JSON.parse(manageAccessTokenResponse.body).access_token;
    } catch(err){
      throw new Error(`When sending a request to get an access token for a Microsoft compliance tenant, an error occured. full error: ${require('util').inspect(err)}`);
    }

    // [?]: https://learn.microsoft.com/en-us/graph/api/resources/serviceprincipal
    let servicePrincipalResponse = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/servicePrincipals?$filter=${encodeURIComponent(`appId eq '0000000a-0000-0000-c000-000000000000'`)}`,
      headers: {
        'Authorization': `Bearer ${graphAccessToken}`
      }
    });

    let servicePrincipalObjectId = servicePrincipalResponse.value[0].id;

    // [?]: https://learn.microsoft.com/en-us/graph/api/group-list-endpoints
    let servicePrincipalEndpointResponse = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/servicePrincipals/${servicePrincipalObjectId}/endPoints`,
      headers: {
        'Authorization': `Bearer ${graphAccessToken}`
      },
    });

    let endpointsInResponse = servicePrincipalEndpointResponse.value;

    let tenantDataSyncService = _.find(endpointsInResponse, {providerName: 'PartnerTenantDataSyncService'});
    if(!tenantDataSyncService) {
      throw new Error(`When sending a request to get the PartnerTenantDataSyncService service principal of a Microsoft compliance tenant, no PartnerTenantDataSyncService service principal was found.`);
    }
    let tenantDataSyncUrl = tenantDataSyncService.uri;

    let deviceDataSyncService = _.find(endpointsInResponse, {providerName: 'PartnerDeviceDataSyncService'});
    if(!deviceDataSyncService) {
      throw new Error(`When sending a request to get the PartnerDeviceDataSyncService service principal of a Microsoft compliance tenant, no PartnerDeviceDataSyncService service principal was found.`);
    }
    let deviceDataSyncUrl = deviceDataSyncService.uri;

    return {
      manageApiAccessToken,
      graphAccessToken,
      tenantDataSyncUrl,
      deviceDataSyncUrl,
    };
  }


};

