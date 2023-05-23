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
    }

  },


  exits: {

  },


  fn: async function ({host}) {

    if(!sails.config.custom.customerWorkspaceOneBaseUrl) {
      throw new Error('No sails.config.custom.customerWorkspaceOneBaseUrl configured! Please set this value to be the base url of the customers Workspace One instance.');
    }

    if(!sails.config.custom.customerWorkspaceOneTenantId) {
      throw new Error('No sails.config.custom.customerWorkspaceOneTenantId configured! Please set this value to be a the "AirWatch" API token from the Customer\'s Workspace One instance.');
    }

    if(!sails.config.custom.customerWorkspaceOneAuthorizationToken) {
      throw new Error('No sails.config.custom.customerWorkspaceOneAuthorizationToken configured! Please set this value to be the authorization header for requests to the customer\'s Workspace One instance.');
    }

    // Send a request to unenroll this host in the customer's Workspace One instance.
    await sails.helpers.http.post.with({
      // Contrary to what you what think the EnterpriseWipe command only unenrolls the host from a Workspace One instance.
      // [?] [Workspace One URL]/API/help/#!/CommandsV1/CommandsV1_ExecuteByAlternateIdAsync
      url: `/api/mdm/devices/commands?searchby=Serialnumber&id=${encodeURIComponent(host.hardware_serial)}&command=EnterpriseWipe`,
      headers: {
        'Authorization': sails.config.custom.customerWorkspaceOneAuthorizationToken,
        'aw-tenant-code': sails.config.custom.customerWorkspaceOneTenantId,
      },
      baseUrl: sails.config.custom.customerWorkspaceOneBaseUrl
    }).intercept('non200Response', (err)=>{
      if(err.raw.statusCode === 404){
        return new Error(`When sending a request to unenroll a host from a Workspace One instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), the specified host was not found on the customer's Workspace One instance. Full error: ${err}`);
      } else if(err.raw.statusCode === 400) {
        return new Error(`When sending a request to unenroll a host from a Workspace One instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), the Workspace One instance could not unenroll the specified host. Full error: ${err}`);
      } else {
        return new Error(`When sending a request to unenroll a host from a Workspace One instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), an error occured. Full error: ${err}`);
      }
    });

    // All done.
    return;
  }


};
