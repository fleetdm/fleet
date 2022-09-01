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


    await NewsletterSubscription.create({emailAddress: emailAddress})
    .tolerate('E_UNIQUE');

    let argins = {};

    // Update the NewsletterSubscription record for this email address with `isSubscribedTo____` boolean attributes based on the subscribeTo input
    if(subscribeTo === 'releases') {
      argins.isSubscribedToReleases = true;
    }
    // FUTURE: Handle more types of subscribeTo inputs

    await NewsletterSubscription.updateOne({emailAddress: emailAddress}).set(argins);

    // All done.
    return;

  }


};
