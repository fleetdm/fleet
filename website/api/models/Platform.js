/**
 * Platform.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    currentUnfrozenGitHubPrNumbers: {
      type: 'json',
      description: 'An array containing the numbers of PRs to the fleetdm/fleet repo that can currently bypass MergeFreeze.',
      example: [13638, 13447, 13673],
      required: true,
    },

    workshopDetails: {
      type: 'json',
      description: 'An array containing the most recent event details formatted to be used on the workshops page',
      required: true,
      defaultsTo: [],
      example: [
        {
          eventTime: 'Jul 13 from 1pm to 5pm PDT',
          eventbriteLink: 'https://www.eventbrite.com/e/gitops-workshop-example-00000000000',
          startsAt: 1783972800000,
          type: 'string',
          workshopAddress: 'string',
          workshopCity: 'string',
        }
      ],
    },


    workshopDetailsLastUpdatedAt: {
      type: 'number',
      required: true,
      defaultsTo: 1,
      description: 'A JS timestamp representing when the event details were last updated.',
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

