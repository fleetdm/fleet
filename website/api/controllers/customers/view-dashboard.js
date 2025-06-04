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
    const stripe = require('stripe')(sails.config.custom.stripeSecret);
    const today = Date.now();
    const oneYearInMs = (1000 * 60 * 60 * 24 * 365);
    const oneYearAgoAt = today - oneYearInMs;
    const thirtyDaysAgoAt = today - (1000 * 60 * 60 * 24 * 30);
    const thirtyDaysFromNowAt = today + (1000 * 60 * 60 * 24 * 30);
    let subscriptionHasBeenRecentlyRenewed = false;
    let subscriptionExpiresSoon = false;
    let subscriptionIsExpired = false;

    // Get subscription Info
    let thisSubscription = await Subscription.findOne({user: this.req.me.id});
    // If the user does not have a subscription, then help them subscribe.
    if(!thisSubscription) {
      throw {redirect: '/customers/new-license'};
    }

    let stripeSubscriptionDetails = await stripe.subscriptions.retrieve(thisSubscription.stripeSubscriptionId);
    let willSubscriptionRenew = true;
    if(stripeSubscriptionDetails.cancel_at_period_end === true){
      willSubscriptionRenew = false;
    }
    // If this subscription is over a year old, and was renewed in the past 30 days set subscriptionHasBeenRecentlyRenewed to true.
    if(thisSubscription.createdAt <= oneYearAgoAt && (thisSubscription.nextBillingAt - oneYearInMs) >= thirtyDaysAgoAt) {
      subscriptionHasBeenRecentlyRenewed = true;
    }

    // If this subscription will renew in the next 30 days, set subscriptionExpiresSoon to true.
    if(thisSubscription.nextBillingAt <= thirtyDaysFromNowAt && willSubscriptionRenew){
      subscriptionExpiresSoon = true;
    }
    // If this subscription is expired, set subscriptionIsExpired to true.
    if(thisSubscription.nextBillingAt <= Date.now()){
      subscriptionIsExpired = true;
    }

    // Respond with view.
    return {
      stripePublishableKey: sails.config.custom.enableBillingFeatures ? sails.config.custom.stripePublishableKey : undefined,
      thisSubscription,
      subscriptionExpiresSoon,
      subscriptionHasBeenRecentlyRenewed,
      willSubscriptionRenew,
      subscriptionIsExpired,
    };

  }


};
