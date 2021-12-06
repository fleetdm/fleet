module.exports = {


  friendlyName: 'Save billing info and subscribe',


  description: '',


  inputs: {

    quoteId: {
      type: 'number',
      required: true,
      description: 'TODO'
    },

    userId: {
      type: 'number',
      required: true,
      description: 'The User account creating this quote'
    },

    newPaymentSource: {
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

    let userRecord = await User.findOne({id: inputs.userId});
    let quoteRecord = await Quote.findOne({id: inputs.quoteId});

    if(!userRecord || !quoteRecord) {
      throw 'notFound';
    }

    let doesUserHaveASubscription = await Subscription.findOne({user: inputs.userId});
    // If this user has a subscription, we'll throw an error.
    if(doesUserHaveASubscription) {
      throw 'forbidden';
    }

    if(!userRecord.stripeCustomerId) {
      let stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
        emailAddress: userRecord.emailAddress
      }).timeout(5000).retry();
      await User.updateOne({id: userRecord.id})
      .set({
        stripeCustomerId
      });
    }

    await sails.helpers.stripe.saveBillingInfo.with({
      stripeCustomerId: userRecord.stripeCustomerId,
      token: inputs.newPaymentSource.stripeToken
    })
    .intercept({ type: 'StripeCardError' }, 'couldNotSaveBillingInfo');

    userRecord = await User.updateOne({ id: inputs.userId })
      .set({
        hasBillingCard: true,
        billingCardBrand: inputs.newPaymentSource.billingCardBrand,
        billingCardLast4: inputs.newPaymentSource.billingCardLast4,
        billingCardExpMonth: inputs.newPaymentSource.billingCardExpMonth,
        billingCardExpYear: inputs.newPaymentSource.billingCardExpYear,
      }).fetch();

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

    // Create the subscription record for this order.
    await Subscription.create({
      organization: userRecord.organization,
      numberOfHosts: quoteRecord.numberOfHosts,
      subscriptionPrice: quoteRecord.quotedPrice,
      status: 'Subscription active',
      quote: quoteRecord.id,
      user: userRecord.id,
      stripeSubscriptionId: subscription.id,
      nextBillingAt: subscription.current_period_end * 1000,
    });

    // Send the order confirmation template email
    await sails.helpers.sendTemplateEmail.with({
      // to: 'eric@feralgoblin.com',
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
