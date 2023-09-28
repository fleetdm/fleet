module.exports = {


  friendlyName: 'Receive Stripe subscription events',


  description: 'Receive events from Stripe about subscription renewals',


  inputs: {
    id: {
      type: 'string',
      description: 'The unique identifier for this Stripe event.',
      moreInfoUrl: 'https://stripe.com/docs/api/events/object#event_object-id',
      required: true,
    },
    type: {
      type: 'string',
      description: 'The type of this Stripe event.',
      moreInfoUrl: 'https://stripe.com/docs/api/events/object#event_object-type',
      required: true,
    },
    data: {
      type: {object: {}},
      description: 'An object containing data associated with this Stripe event.',
      moreInfoUrl: 'https://stripe.com/docs/api/events/object#event_object-data',
      required: true,
    },
    webhookSecret: {
      type: 'string',
      description: 'Used to verify that requests are coming from Stripe.',
      required: true,
    },
  },


  exits: {
    success: { description: 'A Stripe event has successfully been received' },
    missingStripeHeader: { description: 'The webhook received a request with no stripe-signature header', responseType: 'unauthorized'},
  },


  fn: async function ({id, type, data, webhookSecret}) {

    let assert = require('assert');

    if(!this.req.get('stripe-signature')) {
      throw 'missingStripeHeader';
    }

    if (!sails.config.custom.stripeSubscriptionWebhookSecret) {
      throw new Error('No Stripe webhook secret configured!  (Please set `sails.config.custom.stripeSubscriptionWebhookSecret`.)');
    }

    if (sails.config.custom.stripeSubscriptionWebhookSecret !== webhookSecret) {
      throw new Error('Received unexpected Stripe webhook request with webhookSecret set to: '+webhookSecret);
    }

    let stripeEventData = data.object;

    // If this event does not include a subscription ID, we'll ignore it and return a 200 response.
    if(!stripeEventData.subscription) {
      return;
    }
    assert(stripeEventData.customer !== undefined);

    // Find the subscription record for this event.
    let subscriptionIdToFind = stripeEventData.subscription;
    let subscriptionForThisEvent = await Subscription.findOne({stripeSubscriptionId: subscriptionIdToFind}).populate('user');

    let STRIPE_EVENTS_SENT_BEFORE_A_SUBSCRIPTION_RECORD_EXISTS = [
      'invoice.created',
      'invoice.finalized',
      'invoice.paid',
      'invoice.payment_succeeded',
    ];

    // If this event is for a subscription that was just created, we won't have a matching Subscription record in the database. This is because we wait until the subscription's invoice is paid to create the record in our database.
    // To handle cases like this, we'll check to see if a User with the provided stripe customer ID exists, and throw an error if it does not exist.
    if(!subscriptionForThisEvent) {
      if(!_.contains(STRIPE_EVENTS_SENT_BEFORE_A_SUBSCRIPTION_RECORD_EXISTS, type)) {
        throw new Error(`The Stripe subscription events webhook received a event for a subscription with stripeSubscriptionId: ${subscriptionIdToFind}, but no matching record was found in our database.`);
      } else {
        let userReferencedInStripeEvent = await User.findOne({stripeCustomerId: stripeEventData.customer});
        if(!userReferencedInStripeEvent){
          throw new Error(`The receive-from-stripe webhook received an event for an invoice (type: ${type}) for a subscription (stripeSubscriptionId: ${subscriptionIdToFind}) but no matching Subscription or User record (stripeCustomerId: ${stripeEventData.customer}) was found in our databse.`);
        } else {
          return;
        }
      }
    }

    let userForThisSubscription = subscriptionForThisEvent.user;

    // If stripe thinks this subscription renews in 7 days, we'll send the user an subscription reminder email.
    if(type === 'invoice.upcoming' && stripeEventData.billing_reason === 'upcoming') {
      // Get the subscription cost per host for the Subscription renewal notification email.
      let subscriptionCostPerHost = Math.floor(subscriptionForThisEvent.subscriptionPrice / subscriptionForThisEvent.numberOfHosts / 12);
      let upcomingBillingAt = stripeEventData.next_payment_attempt * 1000;
      // Send a upcoming subscription renewal email.
      await sails.helpers.sendTemplateEmail.with({
        to: userForThisSubscription.emailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Your Fleet Premium subscription',
        template: 'email-upcoming-subscription-renewal',
        templateData: {
          firstName: userForThisSubscription.firstName,
          lastName: userForThisSubscription.lastName,
          subscriptionPriceInWholeDollars: subscriptionForThisEvent.subscriptionPrice,
          numberOfHosts: subscriptionForThisEvent.numberOfHosts,
          subscriptionCostPerHost: subscriptionCostPerHost,
          nextBillingAt: upcomingBillingAt,
        }
      });

    } else if(type === 'invoice.paid' && stripeEventData.billing_reason === 'subscription_cycle') {
    // If the event was triggered by a user's card successfully being charged by Stripe, we'll generate a new license key, update the subscription's database record, and send the user a renewal confirmation email.

      if(!stripeEventData.lines || !stripeEventData.lines.data[0]) {
        throw new Error(`When the Stripe subscription events webhook received an event for a paid invoice for subscription id: ${subscriptionIdToFind}, the event data object is missing information about the paid invoice. Check the Stripe dashboard to see the data for this event (Stripe event id: ${id})`);
      }

      // Get the information about the paid invoice from the stripe event.
      let paidInvoiceInformation = stripeEventData.lines.data[0];

      // Convert the new subscription cycle's period end timestamp from Stripe into a JS timestamp.
      let nextBillingAt = paidInvoiceInformation.period.end * 1000;

      // Generate a new license key for this subscription
      let newLicenseKeyForThisSubscription = await sails.helpers.createLicenseKey.with({
        numberOfHosts: subscriptionForThisEvent.numberOfHosts,
        organization: subscriptionForThisEvent.user.organization,
        expiresAt: nextBillingAt,
      });

      // Update the subscription record
      await Subscription.updateOne({id: subscriptionForThisEvent.id}).set({
        fleetLicenseKey: newLicenseKeyForThisSubscription,
        nextBillingAt: nextBillingAt
      });

      // Send subscription renewal email
      await sails.helpers.sendTemplateEmail.with({
        to: userForThisSubscription.emailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Your Fleet Premium subscription',
        template: 'email-subscription-renewal-confirmation',
        templateData: {
          firstName: userForThisSubscription.firstName,
          lastName: userForThisSubscription.lastName,
        }
      });

    }
    // FUTURE: send emails about failed payments. (type === 'invoice.payment_failed' && stripeEventData.billing_reason === 'subscription_cycle')


    return;

  }


};
