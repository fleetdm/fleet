/**
 * ProfileGeneratorUsage.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    credentialType: {
      type: 'string',
      description: 'What this row counts the configuration profile generator usage of: a Fleet Free registration, or a Fleet Premium license.',
      extendedDescription: 'Decides which monthly limits apply.  A registration is created when a Fleet server registers, and a license on its first request.',
      isIn: ['registration', 'license'],
      required: true,
    },

    credentialHash: {
      type: 'string',
      description: 'A SHA-256 hash (hex) that identifies the credential.',
      extendedDescription: 'For a registration, a hash of the secret the Fleet server sends to authenticate its requests.  Only the hash is stored, since the secret spends AI tokens, and the secret is returned to the Fleet server once, when it registers.\n\nFor a license, a hash of the license key\'s header and payload, without its signature.  License keys have no ID claim, and their `sub` (the organization) is not unique, so a license is identified by its signed content.  The signature is left out because the same license can verify under more than one signature encoding.',
      unique: true,
      required: true,
    },

    fleetServerUrl: {
      type: 'string',
      description: 'The URL of the Fleet server that registered.  Empty for a license.',
      extendedDescription: 'Not unique: Fleet servers in development share URLs like http://localhost:8080, and a Fleet server restored from a backup registers again.',
      defaultsTo: '',
    },

    registeredFromIpAddress: {
      type: 'string',
      description: 'The IP address of the client that made a registration.  Empty for a license.',
      extendedDescription: 'Not used yet.  Recorded so that registrations can be throttled per IP address later without a schema change.',
      defaultsTo: '',
    },

    organization: {
      type: 'string',
      description: 'The organization a license was issued to, from its `sub`.  Empty for a registration.',
      extendedDescription: 'Not used by the generator.  Recorded so that support can find a customer\'s usage.',
      defaultsTo: '',
    },

    usageMonth: {
      type: 'string',
      description: 'The UTC month the usage counters on this row belong to, as YYYY-MM.',
      extendedDescription: 'An empty string until the credential is first counted.',
      defaultsTo: '',
    },

    profileGenerationsThisMonth: {
      type: 'number',
      description: 'How many configuration profiles were requested with this credential during its usage month.',
      extendedDescription: 'For a license, this adds up every Fleet server that sends it.',
      defaultsTo: 0,
    },

    nameAndDescriptionRequestsThisMonth: {
      type: 'number',
      description: 'How many configuration profile names and descriptions were requested with this credential during its usage month.',
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
