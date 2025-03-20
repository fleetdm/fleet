module.exports = {


  friendlyName: 'Receive one devices compliance status',


  description: '',


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
    success: { description: 'A devices compliance status was successfully received'},
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
    tenantNotFound: {description: 'No existing Microsoft compliance tenant was found for the Fleet instance that sent the request.', responseType: 'unauthorized'}
  },


  fn: async function ({entraTenantId, fleetServerSecret, deviceId, deviceName, os, osVersion, userId, compliant, lastCheckInTime}) {


    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided entra_tenant_id and fleet_server_secret combination.'});
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = tokenAndApiUrls.accessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;

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
        EasIds: [],
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


    let complianceUpdateResponse = await sails.helpers.http.sendHtttpRequest.with({
      method: 'PUT',
      url: `${deviceDataSyncUrl}/${encodeURIComponent(`DataUploadMessages(guid${sails.helpers.strings.uuid()}`)}?api-version=1.0`,
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


    return {
      message_id: complianceUpdateResponse.MessageId,// eslint-disable-line camelcase
    };
  }


};
