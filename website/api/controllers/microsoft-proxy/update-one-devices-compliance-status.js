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
      required: true,
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
    missingUserPrincipalName: { description: 'A request to update a macOS device\'s complaince status was missing a userPrincipalName value.', responseType: 'badRequest'}
  },


  fn: async function ({entraTenantId, fleetServerSecret, deviceId, deviceManagementState, deviceName, os, osVersion, userPrincipalName, compliant, lastCheckInTime}) {


    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided entra_tenant_id and fleet_server_secret combination.'});
    }

    if(os.toLowerCase() === 'windows') {
      // If this request is for a Windows device, get a graph API token for this tenant to send the compliance status update request.

      let graphTokenResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'POST',
        url: `https://login.microsoftonline.com/${entraTenantId}/oauth2/v2.0/token`,
        enctype: 'application/x-www-form-urlencoded',
        body: {
          client_id: sails.config.custom.compliancePartnerClientId,// eslint-disable-line camelcase
          scope: 'https://graph.microsoft.com/.default',
          client_secret: sails.config.custom.compliancePartnerClientSecret,// eslint-disable-line camelcase
          grant_type: 'client_credentials'// eslint-disable-line camelcase
        }
      });
      let graphAccessToken = JSON.parse(graphTokenResponse.body).access_token;

      // Send the compliance status update.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'PATCH',
        url: `https://graph.microsoft.com/v1.0/devices(deviceId='${encodeURIComponent(deviceId)}')`,
        headers: {
          'Authorization': `Bearer ${graphAccessToken}`
        },
        body: {
          displayName: deviceName,
          operatingSystemVersion: osVersion,
          isManaged: deviceManagementState,
          isCompliant: compliant
        }
      }).intercept((err)=>{
        return new Error({error: `An error occurred when sending a request to sync a Windows device's compliance status for a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
      });

      // Return a 200 response to the Fleet server.
      return {
        message_id: '',// eslint-disable-line camelcase
      };

    } else {
      // If this is a compliance update for a macOS device, we'll need to use the getAccessTokenAndApiUrls helper to get an API URL and token for this request.
      let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
        complianceTenantRecordId: informationAboutThisTenant.id
      });
      let graphAccessToken = tokenAndApiUrls.graphAccessToken;
      let accessToken = tokenAndApiUrls.manageApiAccessToken;
      let deviceDataSyncUrl = tokenAndApiUrls.deviceDataSyncUrl;

      // If a request for a macOS device is missing a userPrincipalName value, return a missingUserPrincipalName (badRequest) response.
      if(!userPrincipalName){
        throw 'missingUserPrincipalName';
      }

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

      let complianceUpdateResponse = await sails.helpers.http.sendHttpRequest.with({
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
      // Log responses from Micrsoft APIs for Fleet's integration
      if(informationAboutThisTenant.fleetInstanceUrl === 'https://dogfood.fleetdm.com') {
        sails.log.info(`Microsoft proxy: update-one-devices-compliance-status sent a compliance update: ${complianceUpdateResponse.body}`);
      }


      return {
        message_id: messageId,// eslint-disable-line camelcase
      };
    }
  }


};
