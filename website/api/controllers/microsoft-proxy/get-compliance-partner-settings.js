module.exports = {


  friendlyName: 'Get compliance partner settings',


  description: '',


  inputs: {

  },


  exits: {
    success: { description: 'The microsoft entra application ID was sent to a managed cloud instance.'},
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
  },


  fn: async function ({}) {
    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    if(!sails.config.custom.entraApplicationId){
      throw new Error(`Missing configuration! PLease set sails.config.custom.entraApplicationId to be the application id of Fleet's microsoft compliance partner application.`)
    }

    // TODO: does this endpoint need to do anything else?

    // All done.
    return {
      entra_application_id: sails.config.custom.entraApplicationId
    };

  }


};
