module.exports = {


  friendlyName: 'Unsubscribe from newsletter',


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
      description: 'This user has successfully unsubscribed from the newsletter.'
    }

  },


  fn: async function ({emailAddress}) {

    let subscription = await NewsletterSubscription.findOne({emailAddress: emailAddress});


    if(!subscription) {
      throw new Error('Consistency violation: Somehow, the NewsletterSubscription record for ' + emailAddress + 'has gone missing');
    } else {
      NewsletterSubscription.updateOne({emailAddress: emailAddress}).set({isActive: false});
    }

    // All done.
    return;

  }


};
