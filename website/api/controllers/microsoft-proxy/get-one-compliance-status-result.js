// TEMPORARY: Shared storage for testing
const testDeviceStorage = require('./_test-device-storage');

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
    success: { description: 'A compliance status update result was returned to the Fleet instance.', outputType: {} },
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

    // Log the status response
    sails.log.info(`Microsoft proxy: Get status result - Status: ${parsedComplianceUpdateResponse.Status}, ErrorCode: ${parsedComplianceUpdateResponse.ErrorCode || 'N/A'}, ErrorDetail: ${parsedComplianceUpdateResponse.ErrorDetail || 'none'}`);

    // If status is "Completed", query Entra ID to verify the compliance status was updated
    if(parsedComplianceUpdateResponse.Status === 'Completed' && testDeviceStorage.lastDeviceId) {
      try {
        let graphAccessToken = tokenAndApiUrls.graphAccessToken;
        let deviceId = testDeviceStorage.lastDeviceId;

        sails.log.info(`Microsoft proxy: Status is Completed - now querying Entra ID for device ${deviceId} to verify compliance update`);

        // Query using $filter on deviceId field (not the object ID)
        let entraDeviceQuery = await sails.helpers.http.get.with({
          url: `https://graph.microsoft.com/v1.0/devices?$filter=${encodeURIComponent(`deviceId eq '${deviceId}'`)}&$select=id,deviceId,displayName,isCompliant,approximateLastSignInDateTime,profileType`,
          headers: {
            'Authorization': `Bearer ${graphAccessToken}`
          }
        });

        if(entraDeviceQuery.value && entraDeviceQuery.value.length > 0) {
          let device = entraDeviceQuery.value[0];
          sails.log.info(`Microsoft proxy: [AFTER COMPLETION] Device info from Entra ID: ${JSON.stringify(device, null, 2)}`);
          sails.log.info(`Microsoft proxy: [AFTER COMPLETION] Entra ID shows device as compliant: ${device.isCompliant}`);
        } else {
          sails.log.warn(`Microsoft proxy: No device found in Entra ID with deviceId ${deviceId}`);
        }
      } catch(err) {
        sails.log.warn(`Microsoft proxy: Failed to query device from Entra ID after completion (non-fatal): ${require('util').inspect(err, {depth: 2})}`);
      }
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
