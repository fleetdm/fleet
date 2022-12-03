/**
 * ExternalAuthorization.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    emailAddress: {
      type: 'string',
      unique: true,
      required: true,
      description: 'The generated source ID used when this external authorization was created.',
    },

    authToken: {
      type: 'string',
      description: 'The token used to authorize requests to this external service.'
    },

    authTokenExpiresAt: {
      type: 'string',
      description: 'A JS timestamp of when the authorization token will expire.'
    },

    refreshToken: {
      type: 'string',
      description: 'The token used to request new authorization tokens from this external service.'
    },

    dataLastSentToVantaAt: {
      type: 'string',
      description: 'A JS Timestamp representing the last time data was sent to Vanta'
    },

    fleetInstanceUrl: {
      type: 'string',
      required: true,
    },

    fleetApiKey: {
      type: 'string',
      required: true,
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

