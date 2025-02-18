module.exports = {


  friendlyName: 'Unsubscribe from all newsletters',


  description: 'Sets the NewsletterSubscription record associated with the provided email address to be inactive',


  inputs: {

    emailAddress: {
      type: 'string',
      description: 'The email address associated with the newsletter subscription that will be set inactive',
      required: true,
    }

  },


  exits: {

    success: {
      description: 'A user has successfully unsubscribed from all newsletter emails.'
    }

  },


  fn: async function ({emailAddress}) {

    let updatedSubscription = await NewsletterSubscription.updateOne({emailAddress: emailAddress}).set({isUnsubscribedFromAll: true});

    if(!updatedSubscription) { // If a subscription could not be found with the specified email address, we'll log a warning and return a 200 response.
      sails.log.warn('When a user tried to unsubscribe from the Fleet newsletter, a NewsletterSubscription record for the specified email address ('+ emailAddress +') could not be found.');
    }
    // All done.
    return this.res.redirect('/#unsubscribed');

  }


};
