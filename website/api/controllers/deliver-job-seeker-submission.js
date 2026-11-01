module.exports = {


  friendlyName: 'Deliver job seeker submission',


  description: 'Send a logged-in user\'s fleetdm.com/hire submission to Fleet.',


  inputs: {

    rolesSought: {
      type: ['string'],
      required: true,
    },

    eligibilityReason: {
      type: 'string',
      required: true,
      isIn: [
        'Fleet customer or former customer',
        'Evaluating Fleet Premium (active PoV)',
        'Fleet-certified (workshop)',
      ],
    },

    eligibilityDetails: {
      type: 'string',
      required: true,
      maxLength: 500,
      description: 'Supporting details for the eligibility claim, e.g. the company that uses Fleet, or the workshop city and date.',
    },

    linkedinUrl: {
      type: 'string',
      required: true,
      maxLength: 500,
    },

    interestedInJobs: {
      type: ['string'],
      defaultsTo: [],
      description: 'Roles on the job board this person is interested in, e.g. "Client Platform Engineer at Acme Inc."',
    },

    notes: {
      type: 'string',
      maxLength: 2000,
    },

  },


  exits: {

    invalidRoles: {
      description: 'One or more of the provided roles is not a role on the Fleet job board.',
      responseType: 'badRequest',
    },

  },


  fn: async function ({rolesSought, eligibilityReason, eligibilityDetails, linkedinUrl, interestedInJobs, notes}) {

    if(rolesSought.length === 0 || _.difference(rolesSought, sails.config.custom.jobBoardRoles).length > 0) {
      throw 'invalidRoles';
    }

    let message =
      `New job seeker on fleetdm.com/hire (verify their eligibility before introducing them): ${this.req.me.firstName} ${this.req.me.lastName} <${this.req.me.emailAddress}>\n`+
      `Claim: ${eligibilityReason}: ${eligibilityDetails}\n`+
      `Looking for: ${rolesSought.join(', ')}\n`+
      `Interested in: ${interestedInJobs.length > 0 ? interestedInJobs.join('; ') : 'n/a'}\n`+
      `LinkedIn: ${linkedinUrl}\n`+
      `Notes: ${notes || 'n/a'}`;

    if(!sails.config.custom.slackWebhookUrlForJobBoard) {
      throw new Error(`Message not delivered: slackWebhookUrlForJobBoard needs to be configured in sails.config.custom. Here's the undelivered message: ${message}`);
    }

    await sails.helpers.http.post.with({
      url: sails.config.custom.slackWebhookUrlForJobBoard,
      data: {text: message},
    })
    .intercept((err) => new Error(`When a user (${this.req.me.emailAddress}) submitted the job board form, Slack could not be notified. Full error: ${require('util').inspect(err, {depth: 2})}`));

  }


};
