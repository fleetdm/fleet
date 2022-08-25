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

    let subscription = await NewsletterSubscription.findOne({emailAddress: emailAddress});


    if(!subscription) {
      throw new Error('Consistency violation: Somehow, the NewsletterSubscription record for ' + emailAddress + 'has gone missing');
    } else {
      NewsletterSubscription.updateOne({emailAddress: emailAddress}).set({isUnsubscribedFromAll: true});
    }

    // All done.
    return;

  }


};
