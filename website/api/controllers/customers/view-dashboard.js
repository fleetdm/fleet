module.exports = {


  friendlyName: 'View dashboard',


  description: 'Display "Dashboard" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/customers/dashboard'
    },

    redirect: {
      description: 'The requesting user does not have a subscription, redirecting to the new license page.',
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



    // Respond with view.
    return {
      stripePublishableKey: sails.config.custom.enableBillingFeatures ? sails.config.custom.stripePublishableKey : undefined,
      thisSubscription,
    };

  }


};
