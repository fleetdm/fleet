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
      description: 'A JS timestamp representing when the event details were last updated.',
    },

    numAiGeneratedProfilesAllTime: {
      type: 'number',
      required: true,
      description: 'The total number of configuration profiles generated with AI, summed from the latest usage statistics report of each Fleet deployment that reported in the past week.',
      extendedDescription: 'Calculated by the send-aggregated-metrics-to-datadog script, since adding up the usage statistics takes too long to do when the profile generator usage page loads.',
    },

    aiGeneratedProfileStatsLastUpdatedAt: {
      type: 'number',
      required: true,
      description: 'A JS timestamp representing when numAiGeneratedProfilesAllTime was last calculated, or 0 if it has never been calculated.',
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

