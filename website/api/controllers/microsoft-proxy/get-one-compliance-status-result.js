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

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = tokenAndApiUrls.manageApiAccessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.deviceDataSyncUrl;

    let complianceStatusResultResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'GET',
      url: `${deviceDataSyncUrl}/DataUploadMessages(guid'${encodeURIComponent(messageId)}')?api-version=1.2`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    }).intercept((err)=>{
      return new Error({error: `An error occurred when retrieving a compliance status result of a device for a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
    });

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
      result.details = parsedComplianceUpdateResponse.ErrorDetail;
    }
    // All done.
    return result;

  }


};
