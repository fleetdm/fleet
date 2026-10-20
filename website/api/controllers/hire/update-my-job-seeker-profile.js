module.exports = {


  friendlyName: 'Update my job seeker profile',


  description: 'Create or update the logged-in user\'s profile on the Fleet job board.',


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
    },

    linkedinUrl: {
      type: 'string',
      required: true,
      maxLength: 500,
    },

    notes: {
      type: 'string',
      maxLength: 2000,
    },

    interestedInJobPostingIds: {
      type: ['number'],
      defaultsTo: [],
    },

    isLooking: {
      type: 'boolean',
      defaultsTo: true,
    },

  },


  exits: {

    success: {
      outputType: {},
    },

    invalidRoles: {
      description: 'One or more of the provided roles is not a role on the Fleet job board.',
      responseType: 'badRequest',
    },

    invalidLinkedinUrl: {
      description: 'The provided LinkedIn URL is not a linkedin.com URL.',
      responseType: 'badRequest',
    },

  },


  fn: async function ({rolesSought, eligibilityReason, eligibilityDetails, linkedinUrl, notes, interestedInJobPostingIds, isLooking}) {

    if(rolesSought.length === 0 || _.difference(rolesSought, sails.config.custom.jobBoardRoles).length > 0) {
      throw 'invalidRoles';
    }

    // Fleet team members click this link, so only allow real LinkedIn URLs (never e.g. "javascript:").
    if(!linkedinUrl.match(/^https?:\/\//i)) {
      linkedinUrl = 'https://' + linkedinUrl;
    }
    let parsedLinkedinUrl;
    try {
      parsedLinkedinUrl = new URL(linkedinUrl);
    } catch (unusedErr) {
      throw 'invalidLinkedinUrl';
    }
    if(!parsedLinkedinUrl.hostname.match(/(^|\.)linkedin\.com$/i)) {
      throw 'invalidLinkedinUrl';
    }
    linkedinUrl = parsedLinkedinUrl.href;

    let existingProfile = await JobSeeker.findOne({user: this.req.me.id});
    let eligibilityClaimChanged = !existingProfile || existingProfile.eligibilityReason !== eligibilityReason || existingProfile.eligibilityDetails !== eligibilityDetails;

    let valuesToSet = {
      rolesSought,
      eligibilityReason,
      eligibilityDetails,
      linkedinUrl,
      notes,
      isLooking,
    };

    if(eligibilityClaimChanged) {
      valuesToSet.verificationStatus = 'Pending';
      // Note: .intercept() can't be chained after .retry() (only the first matching handler runs), so use try/catch.
      try {
        valuesToSet.crmMatchSummary = await sails.helpers.salesforce.getCustomerSummaryForEmailAddress.with({
          emailAddress: this.req.me.emailAddress,
        }).retry();
      } catch (err) {
        throw new Error(`When a user (${this.req.me.emailAddress}) updated their job board profile, their email address could not be looked up in Salesforce. Full error: ${require('util').inspect(err, {depth: 2})}`);
      }
    }

    // Notify Fleet before saving, so that if Slack fails, nothing is saved and resubmitting the form notifies again.
    if(eligibilityClaimChanged && isLooking) {
      let message =
        `New job seeker on fleetdm.com/hire (needs verification): ${this.req.me.firstName} ${this.req.me.lastName} <${this.req.me.emailAddress}>\n`+
        `Claim: ${eligibilityReason}: ${eligibilityDetails}\n`+
        `Salesforce: ${valuesToSet.crmMatchSummary}\n`+
        `Review: ${sails.config.custom.baseUrl}/admin/hire`;
      if(!sails.config.custom.slackWebhookUrlForJobBoard) {
        sails.log.verbose(`slackWebhookUrlForJobBoard is not configured, so this message was not sent to Slack: ${message}`);
      } else {
        try {
          await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForJobBoard, {text: message}).retry();
        } catch (err) {
          throw new Error(`When a user (${this.req.me.emailAddress}) updated their job board profile, Slack could not be notified. Full error: ${require('util').inspect(err, {depth: 2})}`);
        }
      }
    }

    let profile;
    if(existingProfile) {
      profile = await JobSeeker.updateOne({id: existingProfile.id}).set(valuesToSet);
    } else {
      profile = await JobSeeker.create({user: this.req.me.id, ...valuesToSet}).fetch();
    }

    // Only allow interest in roles that are still open.
    let openJobPostingIds = _.pluck(await JobPosting.find({id: {in: interestedInJobPostingIds}, isOpen: true}).select(['id']), 'id');
    await JobSeeker.replaceCollection(profile.id, 'interestedInJobPostings').members(isLooking ? openJobPostingIds : []);

    return _.omit(profile, 'crmMatchSummary');

  }


};
