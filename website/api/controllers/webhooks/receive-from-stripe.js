module.exports = {


  friendlyName: 'Receive Stripe subscription events',


  description: 'Receive events from Stripe about subscription renewals',


  inputs: {
    id: {
      type: 'string',
      required: true,
    },
    type: {
      type: 'string',
      required: true,
    },
    data: {
      type: {object: {}},
      required: true,
    },
    webhookSecret: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'A Stripe event has successfully been received' },
    missingStripeHeader: { description: 'The webhook received a request with no stripe-signature header', responseType: 'unauthorized'},
  },


  fn: async function ({id, type, data, webhookSecret}) {
    const moment = require(sails.config.appPath + '/assets/dependencies/moment.js');

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

    // Find the subscription record for this event.
    let subscriptionIdToFind = stripeEventData.subscription;
    let subscriptionForThisEvent = await Subscription.findOne({stripeSubscriptionId: subscriptionIdToFind}).populate('user');

    if(!subscriptionForThisEvent) {
      throw new Error(`The Stripe subscription events webhook received a event for a subscription with stripeSubscriptionId: ${subscriptionIdToFind}, but no matching record was found in our database.`);
    }

    let userForThisSubscription = subscriptionForThisEvent.user;

    // If this event is an upcoming subscription renewal, we'll send the user an email.
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
          nextBillingAt: moment(upcomingBillingAt).format('MMM Do')+', '+moment(upcomingBillingAt).format('YYYY'),
        }
      });

    } else if(type === 'invoice.paid' && stripeEventData.billing_reason === 'subscription_cycle') {
    // If event is from a Fleet Premium subscription renewal invoice being paid, we'll generate a new license key,
    // update the subscription's database record, and send the user a renewal confirmation email.

      if(!stripeEventData.lines || !stripeEventData.lines.data[0]) {
        throw new Error(`When the Stripe subscription events webhook received an event for a paid invoice for subscription id: ${subscriptionIdToFind}, the event data object is missing information about the paid invoice. Check the Stripe dashboard to see the data for this event (Stripe event id: ${id})`);
      }

      // Get the information about the paid invoice from the stripe event.
      let paidInvoiceInformation = stripeEventData.lines.data[0];

      // Convert the new subscription cycle's period end timestamp from Stripe into a JS timestamp.
      let nextBillingAtInMs = paidInvoiceInformation.period.end * 1000;

      // Generate a new license key for this subscription
      let newLicenseKeyForThisSubscription = await sails.helpers.createLicenseKey.with({
        numberOfHosts: subscriptionForThisEvent.numberOfHosts,
        organization: subscriptionForThisEvent.user.organization,
        expiresAt: nextBillingAtInMs,
      });

      // Update the subscription record
      await Subscription.updateOne({id: subscriptionForThisEvent.id}).set({
        fleetLicenseKey: newLicenseKeyForThisSubscription,
        nextBillingAt: nextBillingAtInMs
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
