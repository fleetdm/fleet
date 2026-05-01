module.exports = {


  friendlyName: 'Get Stripe checkout session url',


  description: 'Creates a Stripe checkout session for a new Fleet Premium subscription and returns the URL',


  inputs: {
    quoteId: {
      type: 'number',
      required: true,
      description: 'The quote to use (determines the price and number of hosts.)'
    },

  },


  exits: {
    success: {
      description: 'A Stripe checkout session was successfully created for a new Fleet Premium subscription.'
    }
  },


  fn: async function (inputs) {

    if (!sails.config.custom.enableBillingFeatures) {// Note: this variable is set in the custom hook if stripePublishableKey and stripeSecret config variables are set.
      throw new Error('The Stripe configuration variables (sails.config.custom.stripePublishableKey and sails.config.custom.stripeSecret) are missing!');
    }

    // Configure Stripe
    const stripe = require('stripe')(sails.config.custom.stripeSecret);

    // Find the quote record that was created.
    let quoteRecord = await Quote.findOne({id: inputs.quoteId});
    if(!quoteRecord) {
      throw new Error(`Consistency violation: The specified quote (${inputs.quoteId}) no longer seems to exist.`);
    }

    let stripeCustomerId = this.req.me.stripeCustomerId;
    // What if the stripe customer id doesn't already exist on the user?
    if (!stripeCustomerId) {
      // Create a new customer entry in the Stripe API for this user before we create a checkout session for their license dispenser purchase.
      stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
        emailAddress: this.req.me.emailAddress
      })
      .timeout(5000)
      .retry()
      .intercept((error)=>{
        return new Error(`An error occurred when trying to create a Stripe Customer for a user (email address: ${this.req.me.emailAddress}) tried to create a Stripe checkout session to purchase a self-service license. Full error: ${error.raw}`);
      });

      await User.updateOne({id: this.req.me.id}).set({stripeCustomerId: stripeCustomerId});
    }
    // Create a new Stripe checkout session for this subscription.
    let stripeCheckoutSession = await stripe.checkout.sessions.create({
      customer: stripeCustomerId,
      customer_update: {// eslint-disable-line camelcase
        name: 'auto',
        address: 'auto',
      },
      success_url: `${sails.config.custom.baseUrl}/customers/dashboard?order-complete`,// eslint-disable-line camelcase
      line_items: [// eslint-disable-line camelcase
        {
          price: sails.config.custom.stripeSubscriptionPriceId,
          quantity: quoteRecord.numberOfHosts,
        },
      ],
      mode: 'subscription',
      billing_address_collection: 'required',// eslint-disable-line camelcase
      allow_promotion_codes: true,// eslint-disable-line camelcase
      tax_id_collection: {// eslint-disable-line camelcase
        enabled: true,
        required: 'if_supported'
      }
    });

    // Return the url of the Stripe checkout session.
    // Users will be taken to this URL via the handleSubmitting function of the <ajax-form> on the /customers/new-license page.
    return stripeCheckoutSession.url;

  }


};
