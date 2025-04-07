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
      requried: true,
    },
    messageId: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    tenantNotFound: {description: 'No existing Microsoft compliance tenant was found for the Fleet instance that sent the request.', responseType: 'unauthorized'}
  },


  fn: async function ({entraTenantId, fleetServerSecret, messageId}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided entra_tenant_id and fleet_server_secret combination.'});
    }

    if(sails.config.custom.sendMockProxyResponsesForDevelopment) {
      sails.log(`Sending mock success response without communicating with the Microsoft API because 'sails.config.custom.sendMockProxyResponsesForDevelopment' is set to true`);
      sails.log(`(Would have returned the result of the a compliance status update sent to Microsoft's API.)`);
      return {
        message_id: messageId,// eslint-disable-line camelcase
        status: 'completed'
      };
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = tokenAndApiUrls.accessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;

    let complianceStatusResultResponse = await sails.helpers.http.sendHtttpRequest.with({
      method: 'GET',
      url: `${deviceDataSyncUrl}/${encodeURIComponent(`DataUploadMessages(guid${messageId}`)}?api-version=1.2`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    }).intercept((err)=>{
      return new Error({error: `An error occurred when retrieving a compliance status result of a device for a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
    });

    let result = {
      message_id: messageId,// eslint-disable-line camelcase
      // status: complianceStatusResultResponse.Status
      status: 'completed'
    };
    // If the status is "Failed", attach the error details to the response body.
    if(complianceStatusResultResponse.Status === 'Failed') {
      result.details = complianceStatusResultResponse.ErrorDetail;
    }
    // All done.
    return result;

  }


};
