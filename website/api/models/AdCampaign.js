/**
 * AdCampaign.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    persona: {
      type: 'string',
      isIn: [
        'elf.it-major-mdm',
        // 'elf.it-gap-filler-mdm',
        // 'elf.it-misc',
        // 'elf.security-vm',
        // 'elf.security-misc',
        // 'santa.it-major-mdm',
        // 'santa.it-gap-filler-mdm',
        // 'santa.it-misc',
        // 'santa.security-vm',
        // 'santa.security-misc',
      ],
      required: true
    },

    name: {
      type: 'string',
      example: 'elf.it-major-mdm - 2024-02-24 @ 6:11pm',
      required: true,
    },

    linkedinCampaignUrn: {
      type: 'string',
      example: 'urn:li:sponsoredCampaign:379399199',
      required: true
    },

    isLatest: {
      type: 'boolean',
      description: 'Whether this is the latest and greatest campaign for this persona.',
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝
    linkedinCompanyIds: {
      type: 'json',
      example: [ 8482494, 28328 ],
      defaultsTo: [],
    },

    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

