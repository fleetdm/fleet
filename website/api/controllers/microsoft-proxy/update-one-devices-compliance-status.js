// TEMPORARY: Shared storage for testing
const testDeviceStorage = require('./_test-device-storage');

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
    tenantNotFound: { description: 'No tenant found matching the credentials', responseType: 'unauthorized' },
    badRequest: { description: 'An error occurred processing the request', responseType: 'badRequest' },
  },


  fn: async function ({entraTenantId, fleetServerSecret, deviceId, deviceManagementState, deviceName, os, osVersion, userPrincipalName, compliant, lastCheckInTime}) {

    // Log the incoming request
    sails.log.info(`Microsoft proxy: Received compliance update request for device ${deviceId} (tenant: ${entraTenantId})`);

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      sails.log.error(`Microsoft proxy: No MicrosoftComplianceTenant record found for tenant ${entraTenantId}`);
      throw 'tenantNotFound';
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });
    let graphAccessToken = tokenAndApiUrls.graphAccessToken;
    let accessToken = tokenAndApiUrls.manageApiAccessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.deviceDataSyncUrl;

    // [?]: https://learn.microsoft.com/en-us/graph/api/resources/users
    // Get the GUID for this user using the UserPrincipalName
    let informationAboutThisUser;
    try {
      informationAboutThisUser = await sails.helpers.http.get.with({
        url: `https://graph.microsoft.com/v1.0/users('${userPrincipalName}')`,
        headers: {
          'Authorization': `Bearer ${graphAccessToken}`
        }
      });
    } catch(err) {
      sails.log.error(`Microsoft proxy: Failed to get user ID from Graph API for ${userPrincipalName}: ${require('util').inspect(err, {depth: 3})}`);
      throw 'badRequest';
    }

    if(!informationAboutThisUser.id) {
      sails.log.error(`Microsoft proxy: Graph API response for user ${userPrincipalName} did not include an ID`);
      throw 'badRequest';
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
        // Using ComplianceState field - Microsoft accepts but doesn't update isCompliant
        // Waiting for Microsoft guidance on correct property naming
        ComplianceState: compliant ? 'compliant' : 'notCompliant'
      }
    ];

    // Generate a UUID for this compliance update.
    let messageId = sails.helpers.strings.uuid();

    // Log the full compliance payload being sent
    sails.log.info(`Microsoft proxy: Sending compliance update for device ${deviceId} (tenant: ${informationAboutThisTenant.entraTenantId})`);
    sails.log.info(`Microsoft proxy: Compliance payload: ${JSON.stringify(complianceUpdateContent, null, 2)}`);

    let complianceUpdateResponse;
    try {
      complianceUpdateResponse = await sails.helpers.http.sendHttpRequest.with({
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
      });
    } catch(err) {
      sails.log.error(`Microsoft proxy: Failed to send compliance update to Microsoft: ${require('util').inspect(err, {depth: 3})}`);
      throw 'badRequest';
    }

    // Log the raw response for debugging
    sails.log.info(`Microsoft proxy: Raw response from Microsoft - Status: ${complianceUpdateResponse.statusCode}, Headers: ${JSON.stringify(complianceUpdateResponse.headers)}, Body length: ${complianceUpdateResponse.body ? complianceUpdateResponse.body.length : 0}`);

    // Check for success status codes (200 OK or 204 No Content)
    if(complianceUpdateResponse.statusCode && complianceUpdateResponse.statusCode !== 200 && complianceUpdateResponse.statusCode !== 204) {
      sails.log.error(`Microsoft proxy: Microsoft returned error status code: ${complianceUpdateResponse.statusCode}. Body: ${complianceUpdateResponse.body}`);
      throw 'badRequest';
    }

    // Extract the message ID from the Location header if present (for 204 responses)
    let operationLocation = complianceUpdateResponse.headers && complianceUpdateResponse.headers.location;
    if(operationLocation) {
      sails.log.info(`Microsoft proxy: Compliance upload accepted. Location: ${operationLocation}`);
    }

    // For 204 responses, there's no body to parse, which is expected
    if(complianceUpdateResponse.statusCode === 204) {
      sails.log.info(`Microsoft proxy: Compliance update successfully accepted by Microsoft (204 No Content)`);
    } else if(complianceUpdateResponse.statusCode === 200) {
      // Parse the upload response for 200 responses
      let parsedComplianceUpdateResponse;
      try {
        if(!complianceUpdateResponse.body || complianceUpdateResponse.body.trim() === '') {
          sails.log.warn(`Microsoft proxy: Got 200 status but empty body`);
        } else {
          parsedComplianceUpdateResponse = JSON.parse(complianceUpdateResponse.body);
          sails.log.info(`Microsoft proxy: Compliance update response: ${JSON.stringify(parsedComplianceUpdateResponse, null, 2)}`);

          // Log OperationLocation if present in the body
          if(parsedComplianceUpdateResponse.OperationLocation) {
            operationLocation = parsedComplianceUpdateResponse.OperationLocation;
            sails.log.info(`Microsoft proxy: OperationLocation from body: ${operationLocation}`);
          }
        }
      } catch(err){
        sails.log.warn(`Microsoft proxy: Failed to parse compliance update response body, but request succeeded. Body: "${complianceUpdateResponse.body}". Error: ${require('util').inspect(err)}`);
      }
    }

    // Store the deviceId for this messageId so we can query Entra later when status is "Completed"
    // We'll query the Entra device in the get-one-compliance-status-result endpoint after Microsoft finishes processing
    testDeviceStorage.lastDeviceId = deviceId;
    sails.log.info(`Microsoft proxy: Compliance upload successful. MessageId: ${messageId}, DeviceId: ${deviceId}`);

    return {
      message_id: messageId,// eslint-disable-line camelcase
    };
  }


};
