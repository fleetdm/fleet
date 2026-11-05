/**
 * ProfileGeneratorRegistration.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    fleetServerUrl: {
      type: 'string',
      description: 'The URL of the Fleet server that registered to use the configuration profile generator.',
      extendedDescription: 'Not unique: Fleet servers in development share URLs like https://localhost:8080, and a Fleet server restored from a backup registers again.',
      required: true,
    },

    fleetServerSecretHash: {
      type: 'string',
      description: 'A SHA-256 hash (hex) of the secret this Fleet server sends to authenticate its profile generation requests.',
      extendedDescription: 'Only the hash is stored, since the secret spends AI tokens.  The secret is returned to the Fleet server once, when it registers.',
      unique: true,
      required: true,
    },

    registeredFromIpAddress: {
      type: 'string',
      description: 'The IP address of the client that made this registration.',
      extendedDescription: 'Not used yet.  Recorded so that registrations can be throttled per IP address later without a schema change.',
      required: true,
    },

    usageMonth: {
      type: 'string',
      description: 'The UTC month the usage counters on this registration belong to, as YYYY-MM.',
      extendedDescription: 'An empty string until this registration first generates a profile.',
      defaultsTo: '',
    },

    profileGenerationsThisMonth: {
      type: 'number',
      description: 'How many configuration profiles this registration has requested during its usage month.',
      defaultsTo: 0,
    },

    nameAndDescriptionRequestsThisMonth: {
      type: 'number',
      description: 'How many configuration profile names and descriptions this registration has requested during its usage month.',
      defaultsTo: 0,
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};
