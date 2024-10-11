/**
 * UndeployedSoftware.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    name: {
      type: 'string',
      required: true,
      description: 'The filename of the software installer package.',
    },

    platform: {
      type: 'string',
      description: 'The type of operating system this software installer is for.',
      required: true,
      isIn: [
        'macOS',
        'Linux',
        'Windows'
      ],
    },

    uploadMime: {
      type: 'string',
      defaultsTo: '',
      description: 'The mime type of the uploaded software installer'
    },

    uploadFd: {
      type: 'string',
      defaultsTo: '',
      description: 'The file descriptor of the installer file.'
    },

    preInstallQuery: {
      type: 'string',
      defaultsTo: '',
    },

    installScript: {
      type: 'string',
      defaultsTo: '',
    },

    postInstallScript: {
      type: 'string',
      defaultsTo: '',
    },

    uninstallScript: {
      type: 'string',
      defaultsTo: '',
    },


    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

