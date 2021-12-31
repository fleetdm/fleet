module.exports = {


  friendlyName: 'Save billing info and subscribe',


  description: '',


  inputs: {

    quoteId: {
      type: 'number',
      required: true,
      description: 'The quote to use (determines the price and number of hosts.)'
    },

    paymentSource: {
      required: true,
      description: 'New payment source info to use (instead of the saved default payment source).',
      extendedDescription: 'If provided, this will also be saved as the new default payment source for the customer, replacing the existing default payment source (if there is one.)',
      type: {
        stripeToken: 'string',
        billingCardLast4: 'string',
        billingCardBrand: 'string',
        billingCardExpMonth: 'string',
        billingCardExpYear: 'string',
      },
      example: {
        stripeToken: 'tok_199k3qEXw14QdSnRwmsK99MH',
        billingCardLast4: '4242',
        billingCardBrand: 'visa',
        billingCardExpMonth: '08',
        billingCardExpYear: '2023',
      }
    },

  },


  exits: {

    couldNotSaveBillingInfo: {
      description: 'The billing information provided could not be saved.',
      responseType: 'badRequest'
    },

  },


  fn: async function (inputs) {
    const stripe = require('stripe')(sails.config.custom.stripeSecret);

    let quoteRecord = await Quote.findOne({id: inputs.quoteId});
    if(!quoteRecord) {
      throw new Error(`Consistency violation: The specified quote (${inputs.quoteId}) no longer seems to exist.`);
    }

    // If this user has a subscription, we'll throw an error.
    let doesUserHaveAnExistingSubscription = await Subscription.findOne({user: this.req.me.id});
    if(doesUserHaveAnExistingSubscription) {
      throw new Error(`Consistency violation: The requesting user (${this.req.me.emailAddress}) already has an existing subscription!`);
    }

    // What if the stripe customer id doesn't already exist on the user?
    // If so, handle this gracefully.  (But why gracefully? But why would this ever be the case?  TODO)
    // let stripeCustomerId;
    // if(this.req.me.stripeCustomerId) {
    //   stripeCustomerId = this.req.me.stripeCustomerId;
    // } else {
    //   stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
    //     emailAddress: this.req.me.emailAddress
    //   }).timeout(5000).retry();

    //   await User.updateOne({id: this.req.me.id})
    //   .set({
    //     stripeCustomerId
    //   });
    // }

    // Write new payment card info ("token") to Stripe's API.
    await sails.helpers.stripe.saveBillingInfo.with({
      stripeCustomerId: this.req.me.stripeCustomerId,
      token: inputs.paymentSource.stripeToken
    })
    .intercept({ type: 'StripeCardError' }, 'couldNotSaveBillingInfo');

    // Save payment card info to our database.
    await User.updateOne({ id: this.req.me.id })
    .set({
      hasBillingCard: true,
      billingCardBrand: inputs.paymentSource.billingCardBrand,
      billingCardLast4: inputs.paymentSource.billingCardLast4,
      billingCardExpMonth: inputs.paymentSource.billingCardExpMonth,
      billingCardExpYear: inputs.paymentSource.billingCardExpYear,
    });

    // Create the subscription for this order in Stripe
    const subscription = await stripe.subscriptions.create({
      customer: this.req.me.stripeCustomerId,
      items: [
        {
          price: sails.config.custom.stripeSubscriptionProduct,
          quantity: quoteRecord.numberOfHosts,
        },
      ],
    });

    // Generate the license key for this subscription
    let licenseKey = await sails.helpers.createLicenseKey.with({
      quoteId: inputs.quoteId,
      validTo: subscription.current_period_end
    });

    // Create the subscription record for this order.
    await Subscription.create({
      organization: this.req.me.organization,
      numberOfHosts: quoteRecord.numberOfHosts,
      subscriptionPrice: quoteRecord.quotedPrice,
      user: this.req.me.id,
      stripeSubscriptionId: subscription.id,
      nextBillingAt: subscription.current_period_end * 1000,
      fleetLicenseKey: licenseKey,
    });

    // Send the order confirmation template email
    await sails.helpers.sendTemplateEmail.with({
      to: this.req.me.emailAddress,
      from: sails.config.custom.fromEmail,
      fromName: sails.config.custom.fromName,
      subject: 'Your Fleet Premium order',
      template: 'email-order-confirmation',
      templateData: {
        firstName: this.req.me.firstName,
        lastName: this.req.me.lastName,
      }
    });

  }


};
