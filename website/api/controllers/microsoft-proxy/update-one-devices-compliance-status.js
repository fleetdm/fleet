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
    deviceManagementState: {
      type: 'boolean',
      description: 'Whether or not this device is enrolled in an MDM.',
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
    userPrincipalName: {
      type: 'string',
      required: true,
    },
    compliant: {
      type: 'boolean',
      required: true,
    },
    lastCheckInTime: {
      type: 'number',
      description: 'The device\'s last sync with Fleet in Unix seconds',
      required: true,
    },
  },


  exits: {
    success: { description: 'A devices compliance status was successfully sent to an entra tenants instance'},
  },


  fn: async function ({entraTenantId, fleetServerSecret, deviceId, deviceManagementState, deviceName, os, osVersion, userPrincipalName, compliant, lastCheckInTime}) {


    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided entra_tenant_id and fleet_server_secret combination.'});
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });
    let graphAccessToken = tokenAndApiUrls.graphAccessToken;
    let accessToken = tokenAndApiUrls.manageApiAccessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.deviceDataSyncUrl;

    // [?]: https://learn.microsoft.com/en-us/graph/api/resources/users
    // Get the GUID for this user using the UserPrincipalName
    let informationAboutThisUser = await sails.helpers.http.get.with({
      url: `https://graph.microsoft.com/v1.0/users('${userPrincipalName}')`,
      headers: {
        'Authorization': `Bearer ${graphAccessToken}`
      }
    }).intercept((err)=>{
      return new Error({error: `An error occurred when getting a user ID from a user principal name (${userPrincipalName}) for a complaince status update. Full error: ${require('util').inspect(err, {depth: 3})}`});
    });

    if(!informationAboutThisUser.id) {
      return new Error({error: `An error occurred when getting information about a user (${userPrincipalName}). The response from the Microsoft graph API did not include an ID.`});
    }

    let lastUpdateTime = new Date().toISOString();

    // Build the compliance report for this device:
    let complianceUpdateContent = [
      {
        EntityType: 1, // EntityType 1 = Device inventory data.
        TenantId: informationAboutThisTenant.entraTenantId,
        DeviceManagementState: deviceManagementState ? 'managed' : 'notManaged',
        DeviceId: deviceId,
        DeviceName: deviceName,
        UserId: informationAboutThisUser.id,
        // LastCheckInTime is a global timestamp indicating the time of device sync with partner service.
        LastCheckInTime: new Date(lastCheckInTime * 1000).toISOString(),
        // LastUpdateTime is a global timestamp indicating the order of messages.
        LastUpdateTime: lastUpdateTime,
        Os: os,
        OsVersion: osVersion,
        EasIds: [],// This field is required but can be sent as an empty array.
        State: 0,
      },
      {
        EntityType: 4, // EntityType 4 = compliance data
        TenantId: informationAboutThisTenant.entraTenantId,
        DeviceId: deviceId,
        UserId: informationAboutThisUser.id,
        // LastUpdateTime is a global timestamp indicating the order of messages.
        LastUpdateTime: lastUpdateTime,
        complianceStatus: compliant ? 'compliant' : 'notCompliant',
      }
    ];

    // Generate a UUID for this compliance update.
    let messageId = sails.helpers.strings.uuid();

    await sails.helpers.http.sendHttpRequest.with({
      method: 'PUT',
      url: `${deviceDataSyncUrl}/DataUploadMessages(guid'${encodeURIComponent(messageId)}')?api-version=1.2`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
      body: {
        TenantId: informationAboutThisTenant.entraTenantId,
        UploadTime: new Date().toISOString(),
        Content: JSON.stringify(complianceUpdateContent),
      }
    }).intercept((err)=>{
      return new Error({error: `An error occurred when sending a request to sync a device's compliance status for a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
    });


    return {
      message_id: messageId,// eslint-disable-line camelcase
    };
  }


};
