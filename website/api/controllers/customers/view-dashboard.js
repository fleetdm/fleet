module.exports = {


  friendlyName: 'View dashboard',


  description: 'Display "Dashboard" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/customers/dashboard'
    },

    redirect: {
      description: 'The requesting user has a subscription, redirecting to the customer dashboard.',
      responseType: 'redirect',
    },

  },


  fn: async function () {

    // Get subscription Info
    let thisSubscription = await Subscription.findOne({user: this.req.me.id});
    // If the user does not have a subscription, then help them subscribe.
    if(!thisSubscription) {
      throw {redirect: '/customers/new-license'};
    }

    // let stripe = require('stripe')(sails.config.custom.stripeSecret);
    // NOTE: leaving this out now 12/03
    // let subscription = await stripe.subscriptions.retrieve(
    //   thisSubscription.stripeSubscriptionId
    // );
    // if (!subscription) {
    //   throw new Error('Stripe thinks this subscription doesnt exist.');
    // }



    // Respond with view.
    return {
      stripePublishableKey: sails.config.custom.enableBillingFeatures? sails.config.custom.stripePublishableKey : undefined,
      thisSubscription,
    };

  }


};
