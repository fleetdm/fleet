module.exports = {


  friendlyName: 'Receive from customer Fleet instance',


  description: 'Receive webhook requests from a customer\'s Fleet instance and sends a request to unenroll a specfied host from the customer\'s Workspace One instance.',


  inputs: {

    timestamp: {
      type: 'string',
      required: true,
      description: 'An ISO 8601 timestamp representing when this request was sent.',
      extendedDescription: 'This value is not used by the webhook.',
      example: '0000-00-00T00:00:00Z'
    },

    host: {
      type: {
        id: 'number',
        uuid: 'string',
        hardware_serial: 'string'//eslint-disable-line camelcase
      },
      required: true,
      description: 'An dictionary containing information about a host that will be unenrolled from the customers Workspace ONE instance.'
    },

    webhookSecret: {
      type: 'string',
      required: true,
      description: 'A shared secret used to confirm that this request came from a customer\'s Fleet instance.',
      extendedDescription: 'This webhook handler should always be requested over TLS.  It is not safe to transmit shared secrets without transport-layer encryption.',
    }

  },


  exits: {
    unauthorized: {
      responseType: 'unauthorized',
      description: 'This webhook request could not be verified.',
    },
    badRequest: {
      responseType: 'badRequest',
      description: 'Invalid MDM migration request.'
    }
  },


  fn: async function ({host, webhookSecret}) {

    if(!sails.config.custom.customerWorkspaceOneBaseUrl) {
      throw new Error('No sails.config.custom.customerWorkspaceOneBaseUrl configured! Please set this value to be the base url of the customers Workspace One instance.');
    }

    if(!sails.config.custom.customerWorkspaceOneOauthId) {
      throw new Error('No sails.config.custom.customerWorkspaceOneOauthId configured! Please set this value to be the client id of the Oauth token for requests to the customer\'s Workspace One instance.');
    }

    if(!sails.config.custom.customerWorkspaceOneOauthSecret) {
      throw new Error('No sails.config.custom.customerWorkspaceOneOauthSecret configured! Please set this value to be the client id of the Oauth token for requests to the customer\'s Workspace One instance.');
    }

    if(!sails.config.custom.customerMigrationWebhookSecret) {
      throw new Error('No sails.config.custom.customerMigrationWebhookSecret configured! Please set this value to be the shared webhook secret for the host migration webhook.');
    }

    if(webhookSecret !== sails.config.custom.customerMigrationWebhookSecret) {
      throw 'unauthorized';
    }

    // Send a request to Workspace ONE to get an authorization token to use for the request to the Workspace ONE instance.
    // [?] https://docs.vmware.com/en/VMware-Workspace-ONE-Access/services/ws1_access_service_administration_cloud/GUID-2B419DC4-7332-448A-9285-E10FF90890F8.html
    // Note: We're using the sendHttpRequest helper so we can set the enctype to be application/x-www-form-urlencoded
    let oauthResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: 'https://na.uemauth.vmwservices.com/connect/token',
      enctype: 'application/x-www-form-urlencoded',
      body: {
        grant_type: 'client_credentials',//eslint-disable-line camelcase
        client_id: sails.config.custom.customerWorkspaceOneOauthId,//eslint-disable-line camelcase
        client_secret: sails.config.custom.customerWorkspaceOneOauthSecret,//eslint-disable-line camelcase
      },
    })
    .intercept((err)=>{
      return new Error(`When sending a request to get a Workspace ONE authorization token for the recieve-from-customer-fleet-instance webhook, an error occured. Full error: ${err.stack}`);
    });

    // The body in responses returned from the sendHttpRequest helper will always be a string, so we need to parse the raw JSON response body to get the access_token returned from the OAuth URL.
    // [?]: https://github.com/sailshq/machinepack-http/blob/9770e75db5f005c21068f5411177bf7e072bfd78/lib/send-http-request.js#L91C27-L94
    let oauthResponseBody = JSON.parse(oauthResponse.body);

    // Send a request to unenroll this host in the customer's Workspace One instance.
    await sails.helpers.http.post.with({
      // Contrary to what you what think the EnterpriseWipe command only unenrolls the host from a Workspace One instance.
      // [?] [Workspace One URL]/API/help/#!/CommandsV1/CommandsV1_ExecuteByAlternateIdAsync
      url: `/api/mdm/devices/commands?searchby=Serialnumber&id=${encodeURIComponent(host.hardware_serial)}&command=EnterpriseWipe`,
      headers: {
        'Authorization': 'Bearer '+oauthResponseBody.access_token,
      },
      baseUrl: sails.config.custom.customerWorkspaceOneBaseUrl
    })
    .intercept('non200Response', (err)=>{
      if(err.raw.statusCode === 404){
        return new Error(`When sending a request to unenroll a host from a Workspace One instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), the specified host was not found on the customer's Workspace One instance. Full error: ${err.stack}`);
      } else if(err.raw.statusCode === 400) {
        return { badRequest: `When sending a request to unenroll a host from a Workspace One instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), the Workspace One instance could not unenroll the specified host. Full error: ${err.stack}` };
      } else {
        return new Error(`When sending a request to unenroll a host from a Workspace One instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), an error occured. Full error: ${err.stack}`);
      }
    });

    // All done.
    return;
  }


};
