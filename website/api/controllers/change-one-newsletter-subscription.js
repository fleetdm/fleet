module.exports = {


  friendlyName: 'Change one newsletter subscription',


  description: '',


  inputs: {
    emailAddress: {
      type: 'string',
      required: true,
    },

    action: {
      type: 'string',
      required: true,
      isIn: ['subscribe', 'unsubscribe']
    },

  },


  exits: {

    success: {
      description: 'A user has successfully changed their subscription to the Fleet newsletter'
    },

  },


  fn: async function ({emailAddress, action}) {

    if(action === 'subscribe') {
      let newSubscriber = await NewsletterSubscription.create({emailAddress: emailAddress});
      //TODO: create Zapier webhook for Salesforce leads
    } else {
      let subscriber = await NewsletterSubscription.findOne({emailAddress: emailAddress});
      if(!subscriber){
        throw new Error('Consistency violation: Somehow, the NewsletterSubscription record for ' + emailAddress + 'has gone missing');
      } else {
        await NewsletterSubscription.updateOne({emailAddress: emailAddress}).set({subscriptionStatus: 'unsubscribed'});
      }
    }

    // All done.
    return;

  }


};
