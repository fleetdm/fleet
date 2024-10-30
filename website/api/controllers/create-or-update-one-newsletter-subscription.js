module.exports = {


  friendlyName: 'Create or update one newsletter subscription',


  description: 'Creates or updates a NewsletterSubscription record for the provided email address',


  inputs: {
    emailAddress: {
      type: 'string',
      required: true,
    },

  },


  exits: {

    success: {
      description: 'A user has successfully changed their subscription to the Fleet newsletter'
    },

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },


  },


  fn: async function ({emailAddress}) {

    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }
    await NewsletterSubscription.create({emailAddress: emailAddress})
    .tolerate('E_UNIQUE');

    await NewsletterSubscription.updateOne({emailAddress: emailAddress}).set({isSubscribedToReleases: true});


    sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress: emailAddress,
      contactSource: 'Website - Newsletter',
      description: `Subscribed to the Fleet newsletter`,
      psychologicalStage: '3 - Intrigued',
      psychologicalStageChangeReason: 'Website - Newsletter',
    }).exec((err)=>{// Use .exec() to run the salesforce helpers in the background.
      if(err) {
        sails.log.warn(`Background task failed: When a user signed up for a newsletter, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
      }
      return;
    });

    // All done.
    return;

  }


};
