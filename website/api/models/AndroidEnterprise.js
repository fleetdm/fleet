/**
 * AndroidEnterprise.js
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
      description: 'The URL of the Fleet server that this Android enterprise exists on.',
      required: true,
    },

    fleetLicenseKey: {
      type: 'string',
      description: 'The license key set on the Fleet server that this Android enterprise exists on.',
    },

    fleetServerSecret: {
      type: 'string',
      description: 'A randomly generated secret used to authenticate requests to an Android proxy endpoint after initial setup.',
      required: true,
    },

    androidEnterpriseId: {
      type: 'string',
      description: 'Google\'s ID for this Android enterprise.',
      extendedDescription: 'This value is set when the Android enterprise is created.'
    },

    pubsubTopicName: {
      type: 'string',
      description: 'The generated pubsub topic name for this Android enterprise',
      extendedDescription: 'This value is saved so we can delete the created pubsub topic if this Android enterprise is deleted.',
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

