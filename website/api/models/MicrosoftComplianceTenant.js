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

    apiKey: {
      type: 'string',
      description: 'The randomly generated API token that will be used to authenticate requests coming from this compliance tenant.'
    },

    entraTenantId: {
      type: 'string',
      description: 'The Microsoft entra tenant ID for this compliance tenant',
      unique: true,
    },

    // TODO: we probably don't need to store this.
    fleetLicenseKey: {
      type: 'string',
      description: 'The license key for the connected Fleet instance'
    },

    fleetInstanceUrl: {
      type: 'string',
      description: 'The url of the connected Fleet instance.'
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

    macosCompliancePolicyGuid: {
      type: 'string',
      description: 'The ID of the compliance policy for macOS devices that was created for this Microsoft tenant.'// TODO: do we need this?
    }

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

