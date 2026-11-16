module.exports = {


  friendlyName: 'Create Google conditional access customer',


  description: 'Registers a Fleet server for a provided Google customer ID and returns a generated secret.',


  inputs: {
    googleCustomerId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'Details about a Google conditional access registration have been returned to a Fleet instance' },
    connectionAlreadyExists: { description: 'The provided Google customer ID is already registered from another Origin.', statusCode: 409 },
    missingOriginHeader: { description: 'No Origin header set', responseType: 'badRequest' },
  },


  fn: async function ({googleCustomerId}) {

    // Google's device APIs use the customer ID without the leading "C", so accept either form and store it with one.
    googleCustomerId = 'C' + googleCustomerId.replace(/^C/, '');

    // Return a badRequest response if the origin header is missing.
    if(!this.req.get('origin')) {// Note: req.get() is case insensitive.
      throw 'missingOriginHeader';
    }

    let fleetServerSecret = sails.helpers.strings.random.with({len: 30});

    let existingRegistration = await GoogleConditionalAccessServer.findOne({googleCustomerId});
    if(existingRegistration) {
      // Only the Fleet server that registered this customer ID can re-create its registration, which replaces the secret.
      if(existingRegistration.fleetInstanceUrl !== this.req.get('origin')) {
        throw 'connectionAlreadyExists';
      }
      await GoogleConditionalAccessServer.updateOne({id: existingRegistration.id}).set({fleetServerSecret});
    } else {
      await GoogleConditionalAccessServer.create({
        fleetServerSecret: fleetServerSecret,
        googleCustomerId: googleCustomerId,
        fleetInstanceUrl: this.req.get('origin'),
      })
      .intercept('E_UNIQUE', 'connectionAlreadyExists');
    }

    return {
      fleetServerSecret: fleetServerSecret,
    };

  }


};
