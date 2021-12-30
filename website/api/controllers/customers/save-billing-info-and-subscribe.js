module.exports = {


  friendlyName: 'Save billing info and subscribe',


  description: '',


  inputs: {

    quoteId: {
      type: 'number',
      required: true,
      description: 'TODO'
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

    forbidden: {
      responseType: 'forbidden',
    },

    notFound: {
      responseType: 'notFound'
    },

    couldNotSaveBillingInfo: {
      description: 'The billing information provided could not be saved.',
      responseType: 'badRequest'
    },

  },


  fn: async function (inputs) {
    const stripe = require('stripe')(sails.config.custom.stripeSecret);

    let userRecord = await User.findOne({id: this.req.me.id});
    let quoteRecord = await Quote.findOne({id: inputs.quoteId});

    if(!userRecord || !quoteRecord) {
      throw 'notFound';
    }

    let doesUserHaveASubscription = await Subscription.findOne({user: userRecord.id});
    // If this user has a subscription, we'll throw an error.
    if(doesUserHaveASubscription) {
      throw 'forbidden';
    }

    if(!userRecord.stripeCustomerId) {
      let stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
        emailAddress: userRecord.emailAddress
      }).timeout(5000).retry();
      userRecord = await User.updateOne({id: userRecord.id})
      .set({
        stripeCustomerId
      });
    }

    await sails.helpers.stripe.saveBillingInfo.with({
      stripeCustomerId: userRecord.stripeCustomerId,
      token: inputs.paymentSource.stripeToken
    })
    .intercept({ type: 'StripeCardError' }, 'couldNotSaveBillingInfo');

    userRecord = await User.updateOne({ id: this.req.me.id })
    .set({
      hasBillingCard: true,
      billingCardBrand: inputs.paymentSource.billingCardBrand,
      billingCardLast4: inputs.paymentSource.billingCardLast4,
      billingCardExpMonth: inputs.paymentSource.billingCardExpMonth,
      billingCardExpYear: inputs.paymentSource.billingCardExpYear,
    });

    // Create the subscription for this order in Stripe
    const subscription = await stripe.subscriptions.create({
      customer: userRecord.stripeCustomerId,
      items: [
        {
          price: sails.config.custom.stripeSubscriptionProduct,
          quantity: quoteRecord.numberOfHosts,
        },
      ],
      // eslint-disable-next-line camelcase
      // trial_period_days: 30,
    });

    // Generate the license key for this subscription;
    let licenseKey = await sails.helpers.createLicenseKey.with({quoteId: inputs.quoteId, validTo: subscription.current_period_end});

    // Create the subscription record for this order.
    await Subscription.create({
      organization: userRecord.organization,
      numberOfHosts: quoteRecord.numberOfHosts,
      subscriptionPrice: quoteRecord.quotedPrice,
      user: userRecord.id,
      stripeSubscriptionId: subscription.id,
      nextBillingAt: subscription.current_period_end * 1000,
      fleetLicenseKey: licenseKey,
    });

    // Send the order confirmation template email
    await sails.helpers.sendTemplateEmail.with({
      to: userRecord.emailAddress,
      from: sails.config.custom.fromEmail,
      fromName: sails.config.custom.fromName,
      subject: 'Your Fleet Premium Order',
      template: 'email-order-confirmation',
      templateData: {
        firstName: userRecord.firstName,
        lastName: userRecord.lastName,
      }
    });


    // All done.
    return;

  }


};
