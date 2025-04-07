/**
 * UndeployedProfile.js
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
      description: 'The name of the configuration profile on the Fleet instance.',
    },

    platform: {
      type: 'string',
      description: 'The type of operating system this profile is for.',
      required: true,
      isIn: [
        'darwin',
        'windows'
      ]
    },

    profileType: {
      type: 'string',
      description: 'The file extension of the configuration profile.',
      required: true,
      isIn: [
        '.mobileconfig',
        '.xml',
        '.json',
      ],
    },

    profileContents: {
      type: 'string',
      required: true,
      description: 'The contents of the configuration profile.',
    },


    labels: {
      type: 'json',
      example: ['All hosts', 'Linux hosts'],
      description: 'A list of the Fleet API IDs of labels this profile is associated with (if any).',
    },

    labelTargetBehavior: {
      type: 'string',
      description: 'Whether to exclude or include hosts with the labels in the labels attribute when assigning this profile.',
      isIn: ['exclude', 'include'],
      defaultsTo: 'include',
    },

    profileTarget: {
      type: 'string',
      description: 'What hosts will be targetted when this profile is deployed. "all" for all hosts, or "custom" if a profile targets hosts by labels',
      isIn: ['all', 'custom'],
      defaultsTo: 'all',
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

