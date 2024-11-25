module.exports = {


  friendlyName: 'Update or create contact and account',


  description: 'Upsert contact±account into Salesforce given fresh data about a particular person, and fresh IQ-enrichment data about the person and account.',


  inputs: {

    // Find by…
    emailAddress: { type: 'string' },
    linkedinUrl: { type: 'string' },

    // Set…
    firstName: { type: 'string'},
    lastName: { type: 'string'},
    organization: { type: 'string' },
    description: { type: 'string' },
    primaryBuyingSituation: { type: 'string' },
    psychologicalStage: {
      type: 'string',
      isIn: [
        '1 - Unaware',
        '2 - Aware',
        '3 - Intrigued',
        '4 - Has use case',
        '5 - Personally confident',
        '6 - Has team buy-in'
      ]
    },
    psychologicalStageChangeReason: {
      type: 'string',
      example: 'Website - Organic start flow'
    },
    contactSource: {
      type: 'string',
      isIn: [
        'Website - Contact forms',
        'Website - Sign up',
        'Website - Newsletter',
      ],
    },
    getStartedResponses: {
      type: 'string',
    },
    intentSignal: {
      type: 'string',
      isIn: [
        'Subscribed to the Fleet newsletter',
        // 'Signed up for a fleetdm.com account',//
        // 'Submitted the "Talk to us" form',
        // 'Submitted the "Send a message" form',
      ],
    }

  },


  exits: {

    success: {
      outputType: {
        salesforceAccountId: 'string',
        salesforceContactId: 'string'
      }
    },

  },

  fn: async function ({emailAddress, linkedinUrl, firstName, lastName, organization, primaryBuyingSituation, psychologicalStage, psychologicalStageChangeReason, contactSource, description, getStartedResponses, intentSignal}) {
    // Return undefined if we're not running in a production environment.
    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return {
        salesforceAccountId: undefined,
        salesforceContactId: undefined
      };
    }

    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    require('assert')(sails.config.custom.iqSecret);
    require('assert')(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS);


    if(!emailAddress && !linkedinUrl){
      throw new Error('UsageError: when updating or creating a contact and account in salesforce, either an email or linkedInUrl is required.');
    }

    // Log in to Salesforce.
    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    let salesforceContactId;
    let salesforceAccountId;

    // Build a dictionary of values we'll update/create a contact record with.
    let valuesToSet = {};
    if(emailAddress){
      valuesToSet.Email = emailAddress;
    }
    if(linkedinUrl){
      valuesToSet.LinkedIn_profile__c = linkedinUrl;// eslint-disable-line camelcase
    }
    if(primaryBuyingSituation) {
      valuesToSet.Primary_buying_situation__c = primaryBuyingSituation;// eslint-disable-line camelcase
    }
    if(getStartedResponses) {
      valuesToSet.Website_questionnaire_answers__c = getStartedResponses;// eslint-disable-line camelcase
    }
    if(description) {
      valuesToSet.Description = description;
    }
    if(intentSignal) {
      valuesToSet.Intent_signals__c = intentSignal;// eslint-disable-line camelcase
    }

    let existingContactRecord;
    // Search for an existing Contact record using the provided email address or linkedIn profile URL.
    if(emailAddress) {
      existingContactRecord = await salesforceConnection.sobject('Contact')
      .findOne({
        Email:  emailAddress,
      });
    } else if(linkedinUrl) {
      existingContactRecord = await salesforceConnection.sobject('Contact')
      .findOne({
        LinkedIn_profile__c: linkedinUrl // eslint-disable-line camelcase
      });
    }

    if(existingContactRecord) {
      // If a description was provided and the contact has a description, append the new description to it.
      if(description && existingContactRecord.Description) {
        valuesToSet.Description = existingContactRecord.Description + '\n' + description;
      }
      // If we're updating a contact, add psychologicalStage and psychologicalStageChangeReason to the dictionary of valuesToSet.
      if(psychologicalStage) {
        valuesToSet.Stage__c = psychologicalStage;// eslint-disable-line camelcase
      }
      if(psychologicalStageChangeReason) {
        valuesToSet.Psystage_change_reason__c = psychologicalStageChangeReason;// eslint-disable-line camelcase
      }
      // If an intent signal was specified, add it to the list of intent signals on the exisitng contact.
      // Note: intent signals values are stored as a single string in salesforce, separated by a semicolon.
      if(intentSignal && existingContactRecord.Intent_signals__c) {
        // Convert the string from the Salesforce record into an array.
        let existingContactIntentSignalsAsAnArray = existingContactRecord.Intent_signals__c.split(';');
        // If this intent signal is not included in the exisitng contacts intent signals, add it.
        if(!existingContactIntentSignalsAsAnArray.includes(intentSignal)) {
          existingContactIntentSignalsAsAnArray.push(intentSignal);
          // Convert the array back into a string to send it to Salesforce.
          valuesToSet.Intent_signals__c = existingContactIntentSignalsAsAnArray.join(';');// eslint-disable-line camelcase
        } else {
          // Otherwise, if the existing contact already has this intent signal tracked, remove it from the valuesToSet
          delete valuesToSet.Intent_signals__c;
        }
      }
      // Check the existing contact record's psychologicalStage.
      if(psychologicalStage) {
        let recordsCurrentPsyStage = existingContactRecord.Stage__c;
        // Because each psychological stage starts with a number, we'll get the first character in the record's current psychological stage and the new psychological stage to make comparison easier.
        let psyStageStageNumberToChangeTo = Number(psychologicalStage[0]);
        let recordsCurrentPsyStageNumber = Number(recordsCurrentPsyStage[0]);
        if(psyStageStageNumberToChangeTo < recordsCurrentPsyStageNumber) {
          // If a psychological stage regression is caused by anything other than the start flow, remove the updated value.
          // This is done to prevent automated psyStage regressions caused by users taking other action on the website. (e.g, Booking a meeting or requesting Fleet swag.)
          if(psychologicalStageChangeReason && psychologicalStageChangeReason !== 'Website - Organic start flow') {
            delete valuesToSet.Stage__c;
            delete valuesToSet.Psystage_change_reason__c;
          }
        }
      }
      // console.log(`Exisitng contact found! ${existingContactRecord.Id}`);
      // If we found an existing contact, we'll update it with the information provided.
      salesforceContactId = existingContactRecord.Id;
      await salesforceConnection.sobject('Contact')
      .update({
        Id: salesforceContactId,
        ...valuesToSet,
      });
      salesforceAccountId = existingContactRecord.AccountId;
      // console.log(`${salesforceContactId} updated!`);
    } else {
      // Otherwise, we'll enrich the information we have, and check for an existing account.
      if(linkedinUrl){
        // If linkedinUrl was provided, strip the protocol and subdomain from the URL.
        linkedinUrl = linkedinUrl.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS, '');
      }
      // Send the information we have to the enrichment helper.
      let enrichmentData = await sails.helpers.iq.getEnriched(emailAddress, linkedinUrl, firstName, lastName, organization);
      // console.log(enrichmentData);
      // Add information from the enrichmentData to the values to set on the new Contact record.
      if(enrichmentData.person && enrichmentData.person.linkedinUrl){
        valuesToSet.LinkedIn_profile__c = enrichmentData.person.linkedinUrl;// eslint-disable-line camelcase
      }
      if(enrichmentData.person && enrichmentData.person.title){
        valuesToSet.Title = enrichmentData.person.title;
      }
      let salesforceAccountOwnerId;
      if(!enrichmentData.employer || !enrichmentData.employer.emailDomain || !enrichmentData.employer.organization) {
        // Special sacrificial meat cave where the contacts with no organization go.
        // https://fleetdm.lightning.force.com/lightning/r/Account/0014x000025JC8DAAW/view
        salesforceAccountId = '0014x000025JC8DAAW';
        salesforceAccountOwnerId = '0054x00000735wDAAQ';// « "Integrations admin" user.
      } else {
        // Search for an existing Account record by the organization returned from the getEnriched helper.
        let existingAccountRecord = await salesforceConnection.sobject('Account')
        .findOne({
          'Name':  enrichmentData.employer.organization,
          // 'LinkedIn_company_URL__c': enrichmentData.employer.linkedinCompanyPageUrl // TODO: if this information is not present on an existing account, nothing will be returned.
        });
        // If we didn't find an account that's name exaclty matches, we'll do another search using the provided email domain.
        if(!existingAccountRecord){
          existingAccountRecord = await salesforceConnection.sobject('Account')
          .findOne({
            'Website':  enrichmentData.employer.emailDomain,
            // 'LinkedIn_company_URL__c': enrichmentData.employer.linkedinCompanyPageUrl // TODO: if this information is not present on an existing account, nothing will be returned.
          });
        }
        // console.log(existingAccountRecord);
        // If we found an exisitng account, we'll assign the new contact to the account owner.
        if(existingAccountRecord) {
          // Store the ID of the Account record we found.
          salesforceAccountId = existingAccountRecord.Id;
          salesforceAccountOwnerId = existingAccountRecord.OwnerId;
          // console.log('exising account found!', salesforceAccountId);
        } else {
          // If no existing account record was found, create a new one, and assign it to the "Integrations Admin" user.
          salesforceAccountOwnerId = '0054x00000735wDAAQ';// « "Integrations admin" user.
          // Create a timestamp to use for the new account's assigned date.
          let today = new Date();
          let nowOn = today.toISOString().replace('Z', '+0000');
          require('assert')(typeof enrichmentData.employer.numberOfEmployees === 'number');
          let newAccountRecord = await salesforceConnection.sobject('Account')
          .create({
            Account_Assigned_date__c: nowOn,// eslint-disable-line camelcase
            // eslint-disable-next-line camelcase
            Current_Assignment_Reason__c: 'Inbound Lead',// TODO verify that this matters. if not, do not set it.
            Prospect_Status__c: 'Assigned',// eslint-disable-line camelcase

            Name: enrichmentData.employer.organization,// IFWMIH: We know organization exists
            Website: enrichmentData.employer.emailDomain,
            LinkedIn_company_URL__c: enrichmentData.employer.linkedinCompanyPageUrl,// eslint-disable-line camelcase
            NumberOfEmployees: enrichmentData.employer.numberOfEmployees,
            OwnerId: salesforceAccountOwnerId
          });
          salesforceAccountId = newAccountRecord.id;
        }//ﬁ
        // console.log('New account created!', salesforceAccountId);
      }//ﬁ

      // Only add contactSource to valuesToSet if we're creating a new contact record.
      if(contactSource) {
        valuesToSet.Contact_source__c = contactSource;// eslint-disable-line camelcase
      }
      // console.log(`creating new Contact record.`)
      // Create a new Contact record for this person.

      let newContactRecord = await salesforceConnection.sobject('Contact')
      .create({
        AccountId: salesforceAccountId,
        OwnerId: salesforceAccountOwnerId,
        FirstName: firstName ? firstName : '?',
        LastName: lastName ? lastName : '?',
        ...valuesToSet,
      });
      salesforceContactId = newContactRecord.id;

      // Since we've created a new contact, we'll update the psychological stage to be either '2 - Aware', or whatever psystage was provided.
      // This causes it to appear as an edit in our CRM and helps reporting.
      await salesforceConnection.sobject('Contact')
      .update({
        Id: salesforceContactId,
        Stage__c: psychologicalStage ? psychologicalStage : '2 - Aware',// eslint-disable-line camelcase
        Psystage_change_reason__c: psychologicalStageChangeReason ? psychologicalStageChangeReason : null,// eslint-disable-line camelcase
      });
      // console.log(`Created ${newContactRecord.id}`);
    }//ﬁ

    return {
      salesforceAccountId,
      salesforceContactId
    };

  }


};

