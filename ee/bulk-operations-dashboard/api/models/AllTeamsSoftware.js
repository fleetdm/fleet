/**
 * AllTeamsSoftware.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝

    fleetApid: {
      type: 'number',
      description: 'The API ID of this software on the connected Fleet instance',
      unique: true,
    },

    teamApids: {
      type: 'json',
      example: [1, 2, 3],
      description: 'An array of the team IDs that this software is deployed on.'
    },

    // preInstallQuery: {
    //   type: 'string',
    //   defaultsTo: '',
    // },

    // installScript: {
    //   type: 'string',
    //   defaultsTo: '',
    // },

    // postInstallScript: {
    //   type: 'string',
    //   defaultsTo: '',
    // },

    // uninstallScript: {
    //   type: 'string',
    //   defaultsTo: '',
    // },


    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

