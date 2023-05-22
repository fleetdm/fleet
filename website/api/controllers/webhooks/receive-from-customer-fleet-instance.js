module.exports = {


  friendlyName: 'Receive from customer fleet instance',


  description: 'Receive webhook requests and/or incoming auth redirects from Customer.',


  inputs: {
    timestamp: {
      type: 'string',
      required: true,
      description: 'An ISO timestamp representing when this request was sent',
    },
    host: {
      type: { id: 'number', uuid: 'string', hardware_serial: 'string' },
      required: true,
      description: 'An object containing information about a host that will be unenrolled from the customers old MDM instance.'
    }
  },


  exits: {

  },


  fn: async function ({timestamp, host}) {

    if(!sails.config.custom.customerWorkspaceOneUrl) {
      throw new Error('No sails.config.custom.customerWorkspaceOneUrl configured! Please set this value to be the base url of the customers MDM instance.')
    }

    if(!sails.config.custom.customerWorkspaceOneTenentID){
      throw new Error('No sails.config.custom.customerWorkspaceOneTenentID configured! Please set this value to be a the "AirWatch" API token from the Customer\'s MDM instance.')//FUTURE: better error
    }

    if(!sails.config.custom.customerWorkspaceOneAuthorizationHeader) {
      throw new Error('No sails.config.custom.customerWorkspaceOneAuthorizationHeader configured! Please set this value to be the authorization header for requests to this customers MDM instance.')//FUTURE: better error
    }

    // Send a request to unenroll this host in the customer's old MDM instance.
    await sails.helpers.http.post.with({
      url: `/api/mdm/devices/commands?searchby=Serialnumber&id=${host.hardware_serial}&command=EnterpriseWipe`,
      headers: {
        'Authorization': sails.config.custom.customerWorkspaceOneAuthorizationHeader,
        'aw-tenant-code': sails.config.custom.customerWorkspaceOneTenentID,
      },
      baseUrl: sails.config.custom.customerWorkspaceOneUrl
    }).intercept((err)=>{
      return new Error(`When sending a request to unenroll a host from an old MDM instance (Host information: Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}), an error occured. Full error: ${err}`);// FUTURE: better error
    })

    // All done.
    return;

  }


};
