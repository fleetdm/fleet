module.exports = {


  friendlyName: 'Redirect to stripe billing portal',


  description: 'Creates a Stripe billing portal session for a Fleet Premium subscriber and redirects them.',


  exits: {
    redirect: {
      responseType: 'redirect',
      description: 'The requesting user is being redirected to the Stripe customer billing portal.'
    },
    noSubscription: {
      responseType: 'redirect',
      description: 'The Requesting user does not have a Fleet premium subscription.'
    },
  },


  fn: async function () {
    // Note: This action is covered by the 'is-logged-in' policy.
    const stripe = require('stripe')(sails.config.custom.stripeSecret);

    let thisUsersSubscription = await Subscription.findOne({user: this.req.me.id});
    if(!thisUsersSubscription){
      throw {noSubscription: '/customers/new-license'};
    }

    let session = await stripe.billingPortal.sessions.create({
      customer: this.req.me.stripeCustomerId,
      return_url: `${sails.config.custom.baseUrl}/customers/dashboard`,// eslint-disable-line camelcase
    });
    // All done.
    throw {redirect: session.url};

  }


};
