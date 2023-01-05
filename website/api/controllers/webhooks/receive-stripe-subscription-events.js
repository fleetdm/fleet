module.exports = {


  friendlyName: 'Receive Stripe subscription events',


  description: 'Receive events from Stripe about subscription renewals',


  exits: {
    success: { description: 'A Stripe event has successfully been received' },
    invalidRequestingIp: {description: 'The webhook received a request from a non stripe IP address', responseType: 'unauthorized'},
    missingStripeHeader: { description: 'The webhook received a request with no stripe-signature header', responseType: 'unauthorized'},
    missingRequestBody: { description: 'The webhook received a request with no body', responseType: 'badRequest'},
    subscriptionUpdated: { description: 'A subscription had been successfully renewed.', responseType: 'ok' },
    subscriptionRenewalEmailSent: { description: 'A user had been sent an email notification of their subscritpion renewal', responseType: 'ok' },
  },


  fn: async function (exits) {
    const stripe = require('stripe');
    const moment = require(sails.config.appPath + '/assets/dependencies/moment.js');
    const VALID_STRIPE_IP_ADDRESSES = [
      '3.18.12.63',
      '3.130.192.231',
      '13.235.14.237',
      '13.235.122.149',
      '18.211.135.69',
      '35.154.171.200',
      '52.15.183.38',
      '54.88.130.119',
      '54.88.130.237',
      '54.187.174.169',
      '54.187.205.235',
      '54.187.216.72'
    ];

    // If the requesting IP address is not in the list of IP addresses that stripe uses, return a
    if(!_.contains(VALID_STRIPE_IP_ADDRESSES, this.req.get('cf-connecting-ip'))){
      throw 'invalidRequestingIp';
    }
    // If this request is missing a stripe-signature header,
    if(!this.req.get('stripe-signature')) {
      throw 'missingStripeHeader';
    }

    const stripeSignatureHeader = this.req.get('stripe-signature');

    if(!this.req.body){
      throw 'missingRequestBody';
    }

    let stripeEvent;

    // Construct a stripe event from the raw request body.
    try {
      stripeEvent = stripe.webhooks.constructEvent(this.req.body, stripeSignatureHeader, sails.config.custom.stripeSubscriptionWebhookSecret);
    } catch (err) {
      // throw an error if there was an error constructing the event.
      throw new Error(`When the webhook received a valid request from Stripe, the event provided could not be constructed by the stripe.webhooks.constructEvent method. Full error ${err}`);
    }

    let stripeEventData = stripeEvent.data.object;

    // If this event has no subscription ID, we'll return a 200 reponse.
    if(!stripeEventData.subscription) {
      return exits.success();
    }

    // Find the subscription record for this event.
    let subscriptionIdToFind = stripeEventData.subscription;
    let subscriptionForThisEvent = await Subscription.findOne({stripeSubscriptionId: subscriptionIdToFind}).populate('User');

    if(!subscriptionForThisEvent){
      throw new Error(`The Stripe subscription events webhook received a event for a subscription with stripeSubscriptionId: ${subscriptionIdToFind}, but no matching record was found in our database.`);
    }

    let userForThisSubscription = subscriptionForThisEvent.user;

    // If this event is an upcoming subscription renewal, we'll send the user an email.
    if(stripeEvent.type === 'invoice.upcoming' && stripeEventData.billing_reason === 'upcoming') {

      // Get the subscription cost per host for the Subscription renewal notification email.
      let subscriptionCostPerHost = Math.floor(subscriptionForThisEvent.subscriptionPrice / subscriptionForThisEvent.numberOfHosts / 12);

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
          nextBillingAt: moment(new Date()).format('MMM Do')+', '+moment(new Date()).format('YYYY'),
        }
      });

      return exits.subscriptionRenewalEmailSent;

    } else if(stripeEvent.type === 'invoice.paid' && stripeEventData.billing_reason === 'subscription_cycle') {
    // If event is from a Fleet Premium subscription renewal invoice being paid, we'll generate a new license key,
    // update the subscription's database record, and send the user a renewal confirmation email.

      // Convert the timestamp from Stripe into a JS timestamp.
      let nextBillingAtInMs = stripeEventData.period_end * 1000;

      // Generate a new license key for this subscription
      let newLicenseKeyForThisSubscription = await sails.helpers.createLicenseKey.with({
        numberOfHosts: subscriptionForThisEvent.numberOfHosts,
        organization: subscriptionForThisEvent.user.organization,
        expiresAt: nextBillingAtInMs,
      });

      // Update the subscription record
      await Subscription.updateOne({id: subscriptionForThisEvent.id}).with({
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

      return exits.subscriptionUpdated;
    }
    // FUTURE: send emails about failed payments. (stripeEvent.type === 'invoice.payment_failed' && stripeEventData.billing_reason === 'subscription_cycle')


    return;

  }


};
