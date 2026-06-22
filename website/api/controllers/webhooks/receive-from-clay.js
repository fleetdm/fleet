module.exports = {


  friendlyName: 'Receive from Clay',


  description: 'Receive webhook requests from Clay.',


  inputs: {
    webhookSecret: {
      type: 'string',
      required: true,
    },

    // For finding/creating contacts.
    firstName: {
      type: 'string',
      required: true,
    },
    lastName: {
      type: 'string',
      required: true,
    },
    linkedinUrl: {
      type: 'string',
    },
    emailAddress: {
      type: 'string',
    },
    contactSource: {
      type: 'string',
      required: true,
      isIn: [
        'Attended a call with Fleet',
        'Event',
        'GitHub - Contributed to fleetdm/fleet',
        'GitHub - Forked fleetdm/fleet',
        'GitHub - Stared fleetdm/fleet',
        'LinkedIn - Comment',
        'LinkedIn - Liked the LinkedIn company page',
        'LinkedIn - Reaction',
        'LinkedIn - Share',
        'Prospecting - AE',
        'Prospecting - Meeting service',
        'Prospecting - Specialist',
        'Website - Chat',
        'Website - Contact forms',
        'Website - Contact forms - Demo - ICP',
        'Website - Contact forms - Demo',
        'Website - GitOps',
        'Website - Newsletter',
        'Website - Sign up',
        'Website - Swag request',
        'Website - Gated document',
        'Webinar',
      ],
    },
    jobTitle: {
      type: 'string',
    },

    // For creating historical event.
    intentSignal: {
      type: 'string',
      required: true,
      isIn: [
        'Followed the Fleet LinkedIn company page',
        'LinkedIn comment',
        'LinkedIn share',
        'LinkedIn reaction',
        'Fleet channel member in MacAdmins Slack',
        'Fleet channel member in osquery Slack',
        'Implemented a trial key',
        'Signed up for Fleet event',
        'Registered for a conference',
        'Engaged with Fleetie at event',
        'Attended a Fleet happy hour',
        'Stared the fleetdm/fleet repo on GitHub',
        'Forked the fleetdm/fleet repo on GitHub',
        'Contributed to the fleetdm/fleet repo on GitHub',
        'Subscribed to the Fleet newsletter',
        'Attended a Fleet training course',
        'Submitted the "Send a message" form',
        'Scheduled a "Talk to us" meeting',
        'Scheduled a "Let\'s get you set up" meeting',
        'Submitted the "GitOps workshop request" form',
        'Signed up for a fleetdm.com account',
        'Requested whitepaper download',
        'Created a quote for a self-service Fleet Premium license',
        'Requested webinar recording',
        'Requested Fleet swag',
      ]
    },
    historicalContent: {
      type: 'string',
      required: true,
    },
    historicalContentUrl: {
      type: 'string',
    },
    relatedCampaign: {
      type: 'string',
    }
  },


  exits: {
    success: { description: 'Information about LinkedIn activity has successfully been received.' },
    duplicateContactOrAccountFound: {description: 'A contact or account could not be created because a duplicate record exists.', statusCode: 409 },
    couldNotCreateContactOrAccount: { description: 'A contact or account could not be created in the CRM using the provided information.' },
    couldNotCreateActivity: { description: 'An error occured when trying to create a historical event record in the CRM' },
  },


  fn: async function ({webhookSecret, firstName, lastName, linkedinUrl, contactSource, jobTitle, intentSignal, historicalContent, historicalContentUrl, relatedCampaign, emailAddress}) {


    if (!sails.config.custom.clayWebhookSecret) {
      throw new Error('No webhook secret configured!  (Please set `sails.config.custom.zapierWebhookSecret`.)');
    }

    if(webhookSecret !== sails.config.custom.clayWebhookSecret){
      throw new Error('Received unexpected webhook request with webhookSecret set to: '+webhookSecret);
    }


    let recordDetails = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      firstName,
      lastName,
      linkedinUrl,
      emailAddress,
      contactSource,
      jobTitle,
    })
    .intercept((err)=>{
      sails.log.warn(`When the receive-from-clay webhook received information about LinkedIn activity, a contact/account could not be created or updated. Full error: ${require('util').inspect(err)}`);
      if(typeof err.errorCode !== 'undefined' && err.errorCode === 'DUPLICATES_DETECTED') {
        return 'duplicateContactOrAccountFound';
      } else {
        return 'couldNotCreateContactOrAccount';
      }
    });

    if(!recordDetails.salesforceAccountId) {
      sails.log.warn(`When the receive-from-clay received information about a user's activity (name: ${firstName} ${lastName}), activity: ${intentSignal}). A contact was successfully updated, but the webhook is unable to continue because this contact is not associated with any Salesforce account record. Contact ID: ${recordDetails.salesforceContactId}`);
      throw 'couldNotCreateActivity';
    }

    let trimmedLinkedinUrl;
    if(linkedinUrl) {
      trimmedLinkedinUrl = linkedinUrl.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS, '');
    }

    // Create the new Fleet website page view record.
    let newHistoricalRecordId = await sails.helpers.salesforce.createHistoricalEvent.with({
      salesforceAccountId: recordDetails.salesforceAccountId,
      salesforceContactId: recordDetails.salesforceContactId,
      eventType: 'Intent signal',
      intentSignal: intentSignal,
      eventContent: historicalContent,
      eventContentUrl: historicalContentUrl,
      linkedinUrl: trimmedLinkedinUrl,
      relatedCampaign: relatedCampaign || recordDetails.mostRecentCampaign,
      eventSource: contactSource,
    })
    .intercept((err)=>{
      sails.log.warn(`When the receive-from-clay webhook received information about LinkedIn activity, a historical event record could not be created. Full error: ${require('util').inspect(err)}`);
      return 'couldNotCreateActivity';
    });

    // All done.
    return {
      historicalRecordId: newHistoricalRecordId,
      contactId: recordDetails.salesforceContactId,
      accountId: recordDetails.salesforceAccountId
    };

  }


};

