/**
 * JobSeeker.js
 *
 * A fleetdm.com user who wants to be introduced to Fleet customers that are hiring.
 * Only visible to Fleet team members (super admins).
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝

    rolesSought: {
      type: 'json',
      defaultsTo: [],
      description: 'The kinds of roles this person is looking for (values from sails.config.custom.jobBoardRoles).',
      example: ['IT Engineer / Client Platform Engineer']
    },

    eligibilityReason: {
      type: 'string',
      required: true,
      isIn: [
        'Fleet customer or former customer',
        'Evaluating Fleet Premium (active PoV)',
        'Fleet-certified (workshop)',
      ],
      description: 'How this person says they qualify for the Fleet job board.'
    },

    eligibilityDetails: {
      type: 'string',
      maxLength: 500,
      description: 'Supporting details for the eligibility claim, e.g. the company that uses Fleet, or the workshop city and date.',
      example: 'GitOps workshop, Denver, 2026-10-03'
    },

    linkedinUrl: {
      type: 'string',
      maxLength: 500,
      example: 'https://www.linkedin.com/in/example'
    },

    notes: {
      type: 'string',
      maxLength: 2000,
      description: 'Anything else this person wants Fleet to know (location, seniority, timing, etc.)'
    },

    isLooking: {
      type: 'boolean',
      defaultsTo: true,
      description: 'Whether this person is still looking. They can turn this off themselves.'
    },

    verificationStatus: {
      type: 'string',
      isIn: ['Pending', 'Verified', 'Not eligible'],
      defaultsTo: 'Pending',
      description: 'Whether a Fleet team member has confirmed this person\'s eligibility claim.',
      extendedDescription: 'Resets to "Pending" whenever the person changes their eligibility claim.'
    },

    crmMatchSummary: {
      type: 'string',
      description: 'What Salesforce knew about this person\'s email address when they submitted, to help Fleet verify their claim.',
      example: 'Contact found at "Acme Inc." (Account type: Customer). 2 closed-won opportunities.'
    },

    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

    user: {
      model: 'User',
      required: true,
    },

    interestedInJobPostings: {
      collection: 'JobPosting',
      via: 'interestedJobSeekers',
    },

  },

};
