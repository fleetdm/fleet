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
    connectionAlreadyExists: { description: 'The provided Google customer ID is already registered.', statusCode: 409 },
    missingOriginHeader: { description: 'No Origin header set', responseType: 'badRequest' },
    invalidOriginHeader: { description: 'The Origin header is not an HTTPS URL.', responseType: 'badRequest' },
  },


  fn: async function ({googleCustomerId}) {

    // Google's device APIs use the customer ID without the leading "C", so accept either form and store it with one.
    googleCustomerId = 'C' + googleCustomerId.replace(/^C/, '');

    // Return a badRequest response if the origin header is missing.
    if(!this.req.get('origin')) {// Note: req.get() is case insensitive.
      throw 'missingOriginHeader';
    }

    // The OAuth callback sends this server's secret to the Origin, so it must be an HTTPS URL for a public hostname (except in local development).
    if(sails.config.environment !== 'development') {
      let origin = this.req.get('origin');
      let parsedOrigin;
      try {
        parsedOrigin = new URL(origin);
      } catch(unusedErr) {
        throw 'invalidOriginHeader';
      }
      let hostname = parsedOrigin.hostname;
      if(parsedOrigin.protocol !== 'https:'
        || parsedOrigin.username || parsedOrigin.password
        || /[?#\\\s]/.test(origin)
        || !hostname.includes('.')
        || hostname.endsWith('.localhost')
        || require('node:net').isIP(hostname) !== 0
      ) {
        throw 'invalidOriginHeader';
      }
    }

    // A customer ID can only be registered once. The Origin header is set by the caller, so a request with the same Origin as
    // the existing registration is not proof that it comes from the Fleet server that registered it, and it can't replace the secret.
    // To register again, a Fleet server deletes its registration first, which requires its secret.
    let existingRegistration = await GoogleConditionalAccessServer.findOne({googleCustomerId});
    if(existingRegistration) {
      throw 'connectionAlreadyExists';
    }

    let fleetServerSecret = require('node:crypto').randomBytes(15).toString('hex');

    await GoogleConditionalAccessServer.create({
      fleetServerSecret: fleetServerSecret,
      googleCustomerId: googleCustomerId,
      fleetInstanceUrl: this.req.get('origin'),
    })
    .intercept('E_UNIQUE', 'connectionAlreadyExists');

    return {
      fleetServerSecret: fleetServerSecret,
    };

  }


};
