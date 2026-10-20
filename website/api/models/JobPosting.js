/**
 * JobPosting.js
 *
 * An open role at a Fleet customer, posted to fleetdm.com/hire by a Fleet team member.
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝

    title: {
      type: 'string',
      required: true,
      maxLength: 200,
      example: 'Senior Client Platform Engineer'
    },

    role: {
      type: 'string',
      required: true,
      description: 'The kind of role this is.',
      extendedDescription: 'One of sails.config.custom.jobBoardRoles. Job seekers pick from the same list.',
      example: 'Security Engineer'
    },

    companyName: {
      type: 'string',
      required: true,
      maxLength: 200,
      example: 'Acme Inc.'
    },

    salesforceAccountId: {
      type: 'string',
      description: 'The ID of the Account record in Salesforce for the hiring company, if it was picked from autocomplete.',
      example: '0014x000025JC8DAAW'
    },

    location: {
      type: 'string',
      maxLength: 200,
      example: 'Remote (US)'
    },

    description: {
      type: 'string',
      maxLength: 5000,
      description: 'A short description of the role.'
    },

    isOpen: {
      type: 'boolean',
      defaultsTo: true,
      description: 'Whether this role is still open and shown on fleetdm.com/hire.'
    },

    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

    postedBy: {
      model: 'User',
      description: 'The Fleet team member who posted this role.'
    },

    interestedJobSeekers: {
      collection: 'JobSeeker',
      via: 'interestedInJobPostings',
    },

  },

};
