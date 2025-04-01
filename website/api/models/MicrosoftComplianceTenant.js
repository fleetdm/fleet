/**
 * MicrosoftComplianceTenant.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝

    fleetServerSecret: {
      type: 'string',
      description: 'The randomly generated API token that will be used to authenticate requests coming from this compliance tenant.'
    },

    entraTenantId: {
      type: 'string',
      description: 'The Microsoft entra tenant ID for this compliance tenant',
      unique: true,
      required: true,
    },

    fleetInstanceUrl: {
      type: 'string',
      description: 'The url of the connected Fleet instance.',
      unique: true,
      required: true,
    },

    setupCompleted: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether or not setups has been completed for this compliance tenant'
    },

    lastHeartbeatAt: {
      type: 'string',
      description: 'A JS timestamp (Epoch MS) representing the last time a heartbeat was sent for this compliance tenant'
    },

    adminConsented: {
      type: 'boolean',
      description: 'Whether or not the an Intune admin consented to add Fleet as a complaince partner.',
      extendedDescription: 'Used only during the initial setup.',
    },

    stateTokenForAdminConsent: {
      type: 'string',
      description: 'A token used to authenticate admin consent webhook requests.',
    }

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

