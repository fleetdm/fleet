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
        'record created',
        'provisioning',
        'ready for assignment',
        'in use',
        'expired',
      ],
      defaultsTo: 'record created',
    },

    slug: {
      type: 'string',
      description: 'The unique slug generated for this Render instance',
      example: 'bumbling-bulbasaur',
      unique: true,
      required: true
    },

    instanceUrl: {
      type: 'string',
      description: 'The full URL of this Fleet instance',
      example: 'https://bumbling-bumblesaur.onrender.com',
    },

    renderProjectId: {
      type: 'string',
      description: 'The ID of the Render project this Fleet instance belongs to.'
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

    renderTrialEndsAt: {
      type: 'number',
      description: 'A JS timestamp representing when the Render isntance associated with this record will be deleted.',
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

    user: {
      model: 'User',
      description: 'The ID of the render POV\'s user. This is not always set because we create a pool oif Render isntances.'
    }

  },

};

