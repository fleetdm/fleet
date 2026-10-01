module.exports = {


  friendlyName: 'Get one compliance status result',


  description: 'Retreives the result of a compliance status update of a Microsoft complaince tenant.',


  inputs: {
    entraTenantId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    messageId: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    success: { description: 'A compliance status update result was returned to the Fleet instance.', outputType: {} },
    unauthorized: { description: 'A request contained an invalid entraTenantId/fleetServerSecret combination.', responseType: 'unauthorized'},
    microsoftApiRequestFailed: {description: 'An error occurred when sending a request to the Microsoft API.'},
    microsoftApiError: {description: 'The Microsoft API returned an unexpected response.'},
  },


  fn: async function ({entraTenantId, fleetServerSecret, messageId}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      throw 'unauthorized';
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    })
    .intercept('microsoftApiRequestFailed', 'microsoftApiRequestFailed')
    .intercept('microsoftApiError', 'microsoftApiError');

    let accessToken = tokenAndApiUrls.manageApiAccessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.deviceDataSyncUrl;

    let complianceStatusResultResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'GET',
      url: `${deviceDataSyncUrl}/DataUploadMessages(guid'${encodeURIComponent(messageId)}')?api-version=1.2`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    })
    .intercept('requestFailed', async ()=>{
      // If a request to the microsoft API fails with a requestFailed error, the cached data sync URL for this tenant may be stale,
      // so clear this tenant's cached tokens and URLs to force re-discovery, and return a microsoftApiRequestFailed response to the Fleet server.
      // The Fleet server retries this request upon error for up to a minute, and if it times out then the host will retry in 1 hour (policy interval).
      await sails.helpers.microsoftProxy.clearCacheForTenant.with({entraTenantId});
      return 'microsoftApiRequestFailed';
    })
    .intercept({raw: {statusCode: 401}}, async (err)=>{
      // If the Microsoft API rejected the cached access token, clear this tenant's cached tokens and URLs to force re-authentication on the next request.
      // The Fleet server retries this request upon error for up to a minute, and if it times out then the host will retry in 1 hour (policy interval).
      await sails.helpers.microsoftProxy.clearCacheForTenant.with({entraTenantId});
      sails.log.warn(`When retrieving a compliance status result of a device for a Microsoft compliance tenant, the cached access token was rejected. Full error: ${require('util').inspect(err, {depth: 3})}`);
      return 'microsoftApiError';
    })
    .intercept((err)=>{
      // If the request to the Microsoft API returns a non-2xx response, log a warning and return a microsoftApiError response
      sails.log.warn(`An error occurred when retrieving a compliance status result of a device for a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`);
      return 'microsoftApiError';
    });

    // Log responses from Micrsoft APIs for Fleet's integration
    if(informationAboutThisTenant.fleetInstanceUrl === 'https://dogfood.fleetdm.com') {
      sails.log.info(`Microsoft proxy: get-one-compliance-status-result retrievied a complaince status result: ${complianceStatusResultResponse.body}`);
    }

    let parsedComplianceUpdateResponse;
    try {
      parsedComplianceUpdateResponse = JSON.parse(complianceStatusResultResponse.body);
    } catch(err){
      throw new Error(`When parsing the JSON response body of a Microsoft compliance partner update status, an error occured. full error: ${require('util').inspect(err)}`);
    }
    let result = {
      message_id: messageId,// eslint-disable-line camelcase
      status: parsedComplianceUpdateResponse.Status
    };
    // If the status is "Failed", attach the error details to the response body.
    if(parsedComplianceUpdateResponse.Status === 'Failed') {
      result.detail = parsedComplianceUpdateResponse.ErrorDetail;
    }
    // All done.
    return result;

  }


};
