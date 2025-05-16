module.exports = {


  friendlyName: 'Receive one devices compliance status',


  description: 'Receives compliance information about a device and sends it to a tenant\'s Entra instance for processing.',


  inputs: {
    entraTenantId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      requried: true,
    },
    deviceId: {
      type: 'string',
      description: 'The devices ID in entra',
      required: true,
    },
    deviceName: {
      type: 'string',
      description: 'The device\'s  display name in Fleet.',
      required: true,
    },
    os: {
      type: 'string',
      required: true,
    },
    osVersion: {
      type: 'string',
      required: true,
    },
    userId: {
      type: 'string',
      required: true,
    },
    compliant: {
      type: 'boolean',
      required: true,
    },
    lastCheckInTime: {
      type: 'number',
      required: true,
    },
  },


  exits: {
    success: { description: 'A devices compliance status was successfully sent to an entra tenants instance'},
  },


  fn: async function ({entraTenantId, fleetServerSecret, deviceId, deviceName, os, osVersion, userId, compliant, lastCheckInTime}) {


    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided entra_tenant_id and fleet_server_secret combination.'});
    }

    // Build the complaince report for this device:
    let complianceUpdateContent = [
      {
        EntityType: 1, // EntityType 1 = Device inventory data.
        TenantId: informationAboutThisTenant.entraTenantId,
        DeviceManagementState: 'managed',
        DeviceId: deviceId,
        DeviceName: deviceName,
        UserId: userId,
        LastCheckInTime: new Date(lastCheckInTime).toISOString(),
        LastUpdateTime: new Date(lastCheckInTime).toISOString(),
        Os: os,
        OsVersion: osVersion,
        EasIds: [],// This field is required but can be sent as an empty array.
        complianceStatus: compliant,
      },
      {
        EntityType: 4, // EntityType 4 = compliance data
        TenantId: informationAboutThisTenant.entraTenantId,
        DeviceId: deviceId,
        UserId: userId,
        LastUpdateTime: new Date(lastCheckInTime).toISOString(),
        complianceStatus: compliant,
      }
    ];

    if(sails.config.custom.sendMockProxyResponsesForDevelopment) {
      sails.log(`Sending mock success response without communicating with the Microsoft API because 'sails.config.custom.sendMockProxyResponsesForDevelopment' is set to true`);
      sails.log(`(Would have sent a compliance status update to microsoft for a host.)`);
      sails.log(`Compliance update content: ${require('util').inspect(complianceUpdateContent, {depth: 3})}`);


      return {
        message_id: sails.helpers.strings.random.with({len:15}),// eslint-disable-line camelcase
      };
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = tokenAndApiUrls.manageApiAccessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.deviceDataSyncUrl;

    let complianceUpdateResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'PUT',
      url: `${deviceDataSyncUrl}/DataUploadMessages(guid'${encodeURIComponent(sails.helpers.strings.uuid())}')?api-version=1.2`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
      body: {
        TenantId: informationAboutThisTenant.entraTenantId,
        UploadTime: new Date().toISOString(),
        Content: complianceUpdateContent,
      }
    }).intercept((err)=>{
      return new Error({error: `An error occurred when sending a request to sync a device's compliance status for a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
    });

    let parsedComplianceUpdateResponse;
    try {
      parsedComplianceUpdateResponse = JSON.parse(complianceUpdateResponse);
    } catch(err){
      throw new Error(`When parsing the JSON response body of a Microsoft compliance partner update, an error occured. full error: ${require('util').inspect(err)}`);
    }

    return {
      message_id: parsedComplianceUpdateResponse.MessageId,// eslint-disable-line camelcase
    };
  }


};
