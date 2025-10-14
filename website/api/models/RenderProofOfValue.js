/**
 * RenderProofOfValue.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    status: {
      type: 'string',
      description: '',
      isIn: [
        'record-created',
        'provisioning',
        'ready-for-assignment',
        'in-use',
        'expiring-soon',
        'expired',
      ],
      defaultsTo: 'record-created',
    },

    slug: {
      type: 'string',
      description: 'The unique slug generated for this Render instance',
      example: 'bumbling-bulbasaur',
      unique: true,
      required: true
    },

    url: {
      type: 'string',
      description: 'The full URL of this Fleet instance',
    },
    renderMySqlServiceId: {
      type: 'string',
      description: 'The ID of the MySQL service this Render POV is configured to use'
    },

    renderRedisServiceId: {
      type: 'string',
      description: 'The ID of the Redis service this Render POV is configured to use'
    },

    renderFleetServiceId: {
      type: 'string',
      description: 'The ID of the Fleet service this Render POV is configured to use'
    },

    renderFleetStorageId: {
      type: 'string',
      description: 'The ID of the disk storage this Render POV is configured to use'
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

