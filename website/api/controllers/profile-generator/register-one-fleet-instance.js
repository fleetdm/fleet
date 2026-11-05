module.exports = {


  friendlyName: 'Register one Fleet instance',


  description: 'Creates a registration for a Fleet server to use the configuration profile generator, and returns the secret it authenticates profile generation requests with.',


  extendedDescription: 'Every call creates a new registration with its own secret and no usage, even for a Fleet server URL that has registered before.',


  inputs: {

    fleetServerUrl: {
      type: 'string',
      required: true,
      description: 'The URL of the Fleet server that is registering.',
      example: 'https://fleet.example.com',
    },

  },


  exits: {

    success: {
      description: 'A Fleet server was registered to use the configuration profile generator.',
      outputType: {
        fleetServerSecret: 'string',
      },
    },

    invalidFleetServerUrl: {
      description: 'The provided Fleet server URL is not an https:// URL.',
      responseType: 'badRequest',
    },

  },


  fn: async function ({fleetServerUrl}) {

    let crypto = require('crypto');

    // Validate the Fleet server URL.
    let parsedUrl;
    try {
      parsedUrl = new URL(fleetServerUrl);
    } catch(unusedErr) {
      throw 'invalidFleetServerUrl';
    }
    if(parsedUrl.protocol !== 'https:') {
      throw 'invalidFleetServerUrl';
    }

    // Generate a secret for this Fleet server.  This uses crypto.randomBytes rather than
    // sails.helpers.strings.random, which is built on Math.random and is not suitable for a credential.
    let fleetServerSecret = crypto.randomBytes(32).toString('hex');

    // Store only a hash of the secret, so that reading the database does not hand out credentials that spend AI tokens.
    await ProfileGeneratorRegistration.create({
      fleetServerUrl,
      fleetServerSecretHash: crypto.createHash('sha256').update(fleetServerSecret).digest('hex'),
      // Behind Heroku's router, this is the client's address because production sets `trustProxy`.
      registeredFromIpAddress: this.req.ip,
    });

    return {
      fleetServerSecret,
    };

  }


};
