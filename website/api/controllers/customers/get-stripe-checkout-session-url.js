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
    // Configure Stripe
    const stripe = require('stripe')(sails.config.custom.stripeSecret);

    // Find the quote record that was created.
    let quoteRecord = await Quote.findOne({id: inputs.quoteId});
    if(!quoteRecord) {
      throw new Error(`Consistency violation: The specified quote (${inputs.quoteId}) no longer seems to exist.`);
    }

    // What if the stripe customer id doesn't already exist on the user?
    if (!this.req.me.stripeCustomerId) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) Stripe customer id has somehow gone missing!`);
    }
    // Create a new Stripe checkout session for this subscription.
    let stripeCheckoutSession = await stripe.checkout.sessions.create({
      customer: this.req.me.stripeCustomerId,
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
