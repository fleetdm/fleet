module.exports = {


  friendlyName: 'Create or update one newsletter subscription',


  description: 'Creates or updates a NewsletterSubscription record for the provided email address',


  inputs: {
    emailAddress: {
      type: 'string',
      required: true,
    },

    subscribeTo: {
      type: 'string',
      required: true,
      description: 'The type of content that this user is changing their subscription for',
      isIn: ['releases']
    }
  },


  exits: {

    success: {
      description: 'A user has successfully changed their subscription to the Fleet newsletter'
    },

  },


  fn: async function ({emailAddress, subscribeTo}) {

    // Look for an existing NewsletterSubscription that uses the provided email address
    let doesSubscriptionForProvidedEmailExist = await NewsletterSubscription.findOne({emailAddress: emailAddress});

    // If one does not exist, we'll create a new one.
    if(!doesSubscriptionForProvidedEmailExist) {
      await NewsletterSubscription.create({emailAddress: emailAddress});
    }

    let argins = {};
    // Once we've found or created a NewsletterSubscription, we'll set the `isSubscribedTo____` boolean attributes based on the subscribeTo input
    if(subscribeTo === 'releases') {
      argins.isSubscribedToReleases = true;
    }
    // FUTURE: Handle more types of subscribeTo inputs

    await NewsletterSubscription.updateOne({emailAddress: emailAddress}).set(argins);

    // All done.
    return;

  }


};
