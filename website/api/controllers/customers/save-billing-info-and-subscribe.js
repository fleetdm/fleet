module.exports = {


  friendlyName: 'Save billing info and subscribe',


  description: '',


  inputs: {

    quoteId: {
      type: 'number',
      required: true,
      description: 'The quote to use (determines the price and number of hosts.)'
    },

    organization: {
      type: 'string',
      description: 'The user\'s organization.'
    },

    firstName: {
      type: 'string',
      description: 'The user\'s first name'
    },

    lastName: {
      type: 'string',
      description: 'The user\'s last name'
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

    cardVerificationRequired: {
      description: 'The billing card provided requires additional verfication before it can be used.',
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
    if (!this.req.me.stripeCustomerId) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) Stripe customer id has somehow gone missing!`);
    }

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
      firstName: inputs.firstName,
      lastName: inputs.lastName,
      organization: inputs.organization,
    });

    // Create the subscription for this order in Stripe
    // [?]: https://stripe.com/docs/api/subscriptions/create?lang=node
    const subscription = await stripe.subscriptions.create({
      customer: this.req.me.stripeCustomerId,
      items: [
        {
          price: sails.config.custom.stripeSubscriptionPriceId,
          quantity: quoteRecord.numberOfHosts,
        },
      ],
    });

    // Get the Stripe ID of the invoice for this subscription.
    let latestInvoiceIdForThisSubscription = subscription.latest_invoice;

    // Get the invoice from Stripe.
    const invoice = await stripe.invoices.retrieve(latestInvoiceIdForThisSubscription);// [?]: https://stripe.com/docs/api/invoices/retrieve?lang=node

    if(!invoice.paid) {
      // If the invoice is not paid, we will throw an error, and ask the customer to contact support.
      // FUTURE: Send an invoice to the customer and update the recieve-from-stripe webhook to handle off-website invoice payments.
      throw 'cardVerificationRequired';
    }

    // Generate the license key for this subscription
    let licenseKey = await sails.helpers.createLicenseKey.with({
      numberOfHosts: quoteRecord.numberOfHosts,
      organization: inputs.organization ? inputs.organization : this.req.me.organization,
      expiresAt: subscription.current_period_end * 1000 // Converting the timestamp from Stripe (in seconds) to a JS timestamp before sending it to the createLicenseKey helper.
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
        firstName: inputs.firstName ? inputs.firstName : this.req.me.firstName,
        lastName: inputs.lastName ? inputs.lastName : this.req.me.lastName,
      }
    });

    let todayOn = new Date();
    let isoTimestampForDescription = todayOn.toISOString();
    sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress: this.req.me.emailAddress,
      firstName: this.req.me.firstName,
      lastName: this.req.me.lastName,
      organization: this.req.me.organization,
      contactSource: 'Website - Sign up',// Note: this is only set on new contacts.
      description: `Purchased a self-service Fleet Premium license on ${isoTimestampForDescription.split('T')[0]} for ${quoteRecord.numberOfHosts} host${quoteRecord.numberOfHosts > 1 ? 's' : ''}.`
    }).exec((err)=>{
      if(err){
        sails.log.warn(`Background task failed: When a user (email: ${this.req.me.emailAddress} purchased a self-service Fleet premium subscription, a Contact and Account record could not be created/updated in the CRM.`, err);
      }
      return;
    });


  }


};
