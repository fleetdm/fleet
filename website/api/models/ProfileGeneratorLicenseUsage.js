/**
 * ProfileGeneratorLicenseUsage.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    licenseKeyHash: {
      type: 'string',
      description: 'A SHA-256 hash (hex) of the Fleet Premium license key\'s header and payload, without its signature.',
      extendedDescription: 'License keys have no ID claim, and their `sub` (the organization) is not unique, so a license is identified by its signed content.  The signature is left out because the same license can verify under more than one signature encoding.',
      unique: true,
      required: true,
    },

    organization: {
      type: 'string',
      description: 'The organization the license was issued to, from its `sub`.',
      extendedDescription: 'Not used by the generator.  Recorded so that support can find a customer\'s usage.',
      defaultsTo: '',
    },

    usageMonth: {
      type: 'string',
      description: 'The UTC month the usage counters on this license belong to, as YYYY-MM.',
      defaultsTo: '',
    },

    profileGenerationsThisMonth: {
      type: 'number',
      description: 'How many configuration profiles Fleet servers with this license have requested during its usage month.',
      defaultsTo: 0,
    },

    nameAndDescriptionRequestsThisMonth: {
      type: 'number',
      description: 'How many configuration profile names and descriptions Fleet servers with this license have requested during its usage month.',
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
