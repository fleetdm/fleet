module.exports = {


  friendlyName: 'Receive one devices compliance status',


  description: '',


  inputs: {
    device_id: {
      type: 'string',
      description: 'The devices ID in entra',
      required: true,
    },
    device_name: {
      type: 'string',
      description: 'The device\'s  display name in Fleet.',
      required: true,
    },
    os: {
      type: 'string',
      required: true,
    },
    os_version: {
      type: 'string',
      required: true,
    },
    user_id: {
      type: 'string',
      required: true,
    }, // Entra ID “user ID”.
    compliant: {
      type: 'boolean',
      required: true,
    },
    last_check_in_time: {
      type: 'number',
      required: true,
    },
    entra_tenant_id: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'A devices compliance status was successfully received'},
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
    tenantNotFound: {description: 'No existing Microsoft compliance tenant was found for the Fleet instance that sent the request.', responseType: 'unauthorized'}
  },


  fn: async function ({device_id, device_name, os, os_version, user_id, compliant, last_check_in_time, entra_tenant_id}) {

    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    let complianceTenantInformation = await MicrosoftComplianceTenant.findOne({entraTenantId: entra_tenant_id});
    if(!complianceTenantInformation) {
      throw 'tenantNotFound';
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
        DeviceId: device_id,
        DeviceName: device_name,
        UserId: user_id,
        LastCheckInTime: new Date(last_check_in_time).toISOString(),
        LastUpdateTime: new Date(last_check_in_time).toISOString(),
        Os: os,
        OsVersion: os_version,
        EasIds: [],
        complianceStatus: compliant,
      },
      {
        EntityType: 4, // EntityType 4 = compliance data
        TenantId: complianceTenantInformation.entraTenantId,
        DeviceId: device_id,
        UserId: user_id,
        LastUpdateTime: new Date(last_check_in_time).toISOString(),
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
