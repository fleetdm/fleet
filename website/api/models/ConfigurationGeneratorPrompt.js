/**
 * ConfigurationGeneratorPrompt.js
 *
 * @description :: A record of a prompt submitted to the configuration generator (fleetdm.com/configuration-generator) and the response.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    profileType: {
      type: 'string',
      required: true,
      isIn: ['mobileconfig', 'csp', 'ddm'],
      description: 'The type of configuration profile the user asked for.',
    },

    prompt: {
      type: 'string',
      required: true,
      columnType: 'LONGTEXT',
      description: 'The natural language instructions the user submitted, stored verbatim.',
    },

    response: {
      type: 'json',
      description: 'The response sent back to the user (the generated profile, or the failure reason), if any.',
    },

    wasSuccessful: {
      type: 'boolean',
      description: 'Whether a configuration profile was successfully generated.',
    },

  },

};
