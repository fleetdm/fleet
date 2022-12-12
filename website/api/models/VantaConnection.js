/**
 * VantaConnection.js
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
      description: 'The email address provided when this Vanta connection was created.',
    },

    fleetInstanceUrl: {
      type: 'string',
      required: true,
      description: 'The full URL of the Fleet instance that will be connected to Vanta.'
    },

    fleetApiKey: {
      type: 'string',
      required: true,
      description: 'The token used to authenticate requests to the user\'s Fleet instance.',
      extendedDescription: 'This token must be for an API-only user and needs to have admin privileges on the user\'s Fleet instance'
    },

    vantaToken: {
      type: 'string',
      description: 'The token used to authorize requests to this external service.'
    },

    vantaTokenExpiresAt: {
      type: 'number',
      description: 'A JS timestamp of when the authorization token will expire.'
    },

    vantarefreshToken: {
      type: 'string',
      description: 'The token used to request new authorization tokens from this external service.'
    },

    dataLastSentToVantaAt: {
      type: 'number',
      description: 'A JS timestamp representing the last time data was sent to Vanta'
    },

    isConnectedToVanta: {
      type: 'boolean',
      description: 'whether this external connection is authorized to send data to Vanta on behalf of the user.',
      extendedDescription: 'This value is set to false if the automated `send-data-to-vanta` script encounters an error when sending data to Vanta.'
    },

    lastErrorAboutThisConnection: {
      type: 'json',
      description: 'The last error that was logged regarding this Vanta connection',
      extendedDescription: 'This information can be used to diagnose why a isConnectedToVanta value is false'
    }

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

