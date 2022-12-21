/**
 * VantaConnection.js
 *
 * @description :: An organization who is a customer of Vanta.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    emailAddress: {
      description: 'The email address provided when this Vanta connection was created.',
      extendedDescription: 'This will be used to contact the user who created this request if any problems occur.',
      type: 'string',
      required: true,
      isEmail: true,
    },

    vantaSourceId: {
      description: 'The generated source ID that will be used the identifier for this for this Vanta Connection in Vanta.',
      type: 'string',
      unique: true,
      required: true,
    },

    fleetInstanceUrl: {
      description: 'The full URL of the Fleet instance that will be connected to Vanta.',
      example: 'https://fleet.example.com',
      type: 'string',
      required: true,
      unique: true,
    },

    fleetApiKey: {
      type: 'string',
      required: true,
      description: 'The token used to authenticate requests to the user\'s Fleet instance.',
      extendedDescription: 'This token must be for an API-only user and needs to have admin privileges on the user\'s Fleet instance'
    },

    vantaAuthToken: {
      type: 'string',
      description: 'A token used to authorize requests to Vanta on behalf of this Vanta customer.'
    },

    vantaAuthTokenExpiresAt: {
      type: 'number',
      description: 'A JS timestamp of when this connection\'s authorization token will expire.'
    },

    vantaRefreshToken: {
      type: 'string',
      description: 'The token used to request new authorization tokens for this Vanta connection.'
    },

    isConnectedToVanta: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether this Vanta connection has been authorized to send data to Vanta on behalf of the user.',
      extendedDescription: 'If this value is true, data from the Fleet instance associated with this connection be sent to Vanta in the send-data-to-vanta script.'
    }

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

