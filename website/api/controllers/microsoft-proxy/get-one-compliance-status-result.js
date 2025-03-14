module.exports = {


  friendlyName: 'Get one compliance status result',


  description: '',


  inputs: {
    message_id: {
      type: 'string',
      required: true,
    },
    entra_tenant_id: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
    tenantNotFound: {description: 'No existing Microsoft compliance tenant was found for the Fleet instance that sent the request.', responseType: 'unauthorized'}
  },


  fn: async function ({message_id, entra_tenant_id}) {
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

    let complianceStatusResultResponse = await sails.helpers.http.sendHtttpRequest.with({
      method: 'GET',
      url: `${deviceDataSyncUrl}/${encodeURIComponent(`DataUploadMessages(guid${message_id}`)}?api-version=1.2`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });
    let result = {
      message_id,
      status: complianceStatusResultResponse.Status
    };

    if(complianceStatusResultResponse.Status === 'Failed') {
      result.details = complianceStatusResultResponse.ErrorDetail;
    }
    // All done.
    return result;

  }


};
