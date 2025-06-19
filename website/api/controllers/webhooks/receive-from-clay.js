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
      required: true,
    },
    contactSource: {
      type: 'string',
      required: true
    },
    jobTitle: {
      type: 'string',
    },

    // For creating historical event.
    intentSignal: {
      type: 'string',
      required: true,
    },
    historicalContent: {
      type: 'string',
      required: true,
    },
    historicalContentUrl: {
      type: 'string',
    }
  },


  exits: {
    success: { description: 'Information about LinkedIn activity has successfully been received.' },
    duplicateContactOrAccountFound: {description: 'A contact or account could not be created because a duplicate record exists.', statusCode: 409 },
    couldNotCreateContactOrAccount: { description: 'A contact or account could not be created in the CRM using the provided information.' },
    couldNotCreateActivity: { description: 'An error occured when trying to create a historical event record in the CRM' },
  },


  fn: async function ({webhookSecret, firstName, lastName, linkedinUrl, contactSource, jobTitle, intentSignal, historicalContent, historicalContentUrl}) {


    if (!sails.config.custom.clayWebhookSecret) {
      throw new Error('No webhook secret configured!  (Please set `sails.config.custom.zapierWebhookSecret`.)');
    }

    if(webhookSecret !== sails.config.custom.clayWebhookSecret){
      throw new Error('Received unexpected webhook request with webhookSecret set to: '+webhookSecret);
    }


    let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      firstName,
      lastName,
      linkedinUrl,
      contactSource,
      jobTitle,
    }).intercept((err)=>{
      sails.log.warn(`When the receive-from-clay webhook received information about LinkedIn activity, a contact/account could not be created or updated. Full error: ${require('util').inspect(err)}`);
      if(typeof err.errorCode !== 'undefined' && err.errorCode === 'DUPLICATES_DETECTED') {
        return 'duplicateContactOrAccountFound';
      } else {
        return 'couldNotCreateContactOrAccount';
      }
    });

    if(!recordIds.salesforceAccountId) {
      sails.log.warn(`When the receive-from-clay received information about a user's activity (name: ${firstName} ${lastName}), activity: ${intentSignal}). A contact was successfully updated, but the webhook is unable to continue because this contact is not associated with any Salesforce account record. Contact ID: ${recordIds.salesforceContactId}`)
      throw 'couldNotCreateActivity';
    }

    let trimmedLinkedinUrl = linkedinUrl.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS, '');

    // Create the new Fleet website page view record.
    let newHistoricalRecordId = await sails.helpers.salesforce.createHistoricalEvent.with({
      salesforceAccountId: recordIds.salesforceAccountId,
      salesforceContactId: recordIds.salesforceContactId,
      eventType: 'Intent signal',
      intentSignal: intentSignal,
      eventContent: historicalContent,
      eventContentUrl: historicalContentUrl,
      linkedinUrl: trimmedLinkedinUrl,
    }).intercept((err)=>{
      sails.log.warn(`When the receive-from-clay webhook received information about LinkedIn activity, a historical event record could not be created. Full error: ${require('util').inspect(err)}`);
      return 'couldNotCreateActivity';
    });

    // All done.
    return {
      historicalRecordId: newHistoricalRecordId,
      contactId: recordIds.salesforceContactId,
      accountId: recordIds.salesforceAccountId
    };

  }


};
