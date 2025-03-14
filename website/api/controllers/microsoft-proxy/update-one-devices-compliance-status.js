module.exports = {


  friendlyName: 'Receive one devices compliance status',


  description: '',


  inputs: {
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
    }, // Entra ID “user ID”.
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


  fn: async function ({deviceId, deviceName, os, osVersion, userId, compliant, lastCheckInTime}) {

    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    if(!this.req.headers['Authorization']) {
      return this.res.unauthorized();
    }
    let authHeaderValue = this.req.headers['Authorization'];
    let tokenForThisRequest = authHeaderValue.split('Bearer ')[1];
    let complianceTenantInformation = await MicrosoftComplianceTenant.findOne({apiKey: tokenForThisRequest});
    if(!complianceTenantInformation) {
      return this.res.notFound();
    }


    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: complianceTenantInformation.id
    });

    let accessToken = tokenAndApiUrls.accessToken;
    let deviceDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;

    // Build the report for thsi device:
    let complianceUpdateContent = [
      {
        EntityType: 1, // EntityType 1 = Device inventory data.
        TenantId: complianceTenantInformation.entraTenantId,
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
        TenantId: complianceTenantInformation.entraTenantId,
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
        TenantId: complianceTenantInformation.entraTenantId,
        UploadTime: new Date().toISOString(),
        Content: complianceUpdateContent,
      }
    });

    // Create a database record for this message.
    let newMessageRecord = await MicrosoftComplianceStatusMessage.create({messageId: complianceUpdateResponse.MessageId});

    return {
      message_id: newMessageRecord.messageId,
    };
  }


};
