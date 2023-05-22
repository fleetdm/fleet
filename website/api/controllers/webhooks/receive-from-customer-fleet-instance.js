module.exports = {


  friendlyName: 'Receive from customer fleet instance',


  description: 'Receive webhook requests and/or incoming auth redirects from Customer.',


  inputs: {
    timestamp: {
      type: 'string',
      required: true,
      description: 'TODO',
    },
    host: {
      type: { id: 'number', uuid: 'string', hardware_serial: 'string' },
      required: true,
      description: 'TODO'
    }
  },


  exits: {

  },


  fn: async function (timestamp, host) {

    if(!sails.config.custom.customerWorkspaceOneUrl) {
      throw new Error('No sails.config.custom.customerWorkspaceOneUrl configured! Please set this!')// FUTURE: better error
    }

    if(!sails.config.custom.customerWorkspaceOneTenentID){
      throw new Error('No sails.config.custom.customerWorkspaceOneTenentID configured! Please set this!')//FUTURE: better error
    }

    if(!sails.config.custom.customerWorkspaceOneAuthorizationHeader) {
      throw new Error('No sails.config.custom.customerWorkspaceOneAuthorizationHeader configured! Please set this!')//FUTURE: better error
    }

    // Send request to old MDM instance:
    await sails.helpers.http.post.with({
      url: `/api/mdm/devices/commands?searchby=Serialnumber&id=${host.hardware_serial}&command=EnterpriseWipe`,
      headers: {
        'Authorization': sails.config.custom.customerWorkspaceOneAuthorizationHeader,
        'aw-tenant-code': sails.config.custom.customerWorkspaceOneTenentID,
      },
      baseUrl: sails.config.custom.customerWorkspaceOneUrl
    }).intercept((err)=>{
      return new Error(`When sending a request to unenroll a host from (Serial number: ${host.hardware_serial}, id: ${host.id}, uuid: ${host.uuid}) a Workspace one instance, an error occured. Full error: ${err}`);// FUTURE: better error
    })



    // All done.
    return;

  }


};
