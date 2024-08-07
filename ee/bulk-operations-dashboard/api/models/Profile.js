/**
 * Profile.js
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
      description: 'The name of the profile on the Fleet instance',
    },

    uuid: {
      type: 'string',
      required: true,
      description: 'The uuid of the profile on the Fleet instance',
    },

    uploadedAt: {
      type: 'string',
      required: true,
      description: 'A JS timestamp of when this profile was uploaded.'
    },

    teamApid: {
      type: 'number',
      description: 'The Fleet API ID of the team this profile is on.',
      extendedDescription: 'Set to 0 if this script is on the "no team" team or undefined if the profile is not on a team'
    },

    teamDisplayName: {
      type: 'string',
      description: 'The name of the team this profile is on.'
    },

    platform: {
      type: 'string',
      description: 'The type of operating system this profile is for.',

    },

    profileType: {// ∆: Do we need this attribute?
      type: 'string',
      description: 'The filestype of the profile',
      isIn: [
        '.mobileconfig',
        '.xml',
      ],
    },

    profileContents: {// ∆: This may
      type: 'ref',
      description: 'The contents of the profile.', // ∆: improve description
      extendedDescription: 'This attribute will only be present on undeployed profiles.'
    },


    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

