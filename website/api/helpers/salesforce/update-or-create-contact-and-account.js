module.exports = {


  friendlyName: 'Update or create contact and account',


  description: 'Upsert contact±account into Salesforce given fresh data about a particular person, and fresh IQ-enrichment data about the person and account.',


  inputs: {

    // Find by…
    emailAddress: { type: 'string' },
    linkedinUrl: { type: 'string' },

    // Set…
    firstName: { type: 'string', required: true },
    lastName: { type: 'string', required: true },
    organization: { type: 'string' },
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
  },


  exits: {

    success: {
      outputType: {
        salesforceAccountId: 'string',
        salesforceContactId: 'string'
      }
    },

  },


  fn: async function ({emailAddress, linkedinUrl, firstName, lastName, organization, primaryBuyingSituation, psychologicalStage}) {
    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return;
    }

    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    require('assert')(sails.config.custom.iqSecret);


    if(!emailAddress && !linkedinUrl){
      throw new Error('UsageError: when updating or creating a contact and account in salesforce, either an email or linkedInUrl is required.');
    }
    // Send the information we have to the enrichment helper.
    let enrichmentData = await sails.helpers.iq.getEnriched(emailAddress, linkedinUrl, firstName, lastName, organization);
    // console.log(enrichmentData);

    // Log in to Salesforce.
    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });

    let salesforceAccountOwnerId;

    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    let salesforceAccountId;
    if(!organization && !enrichmentData.employer){
      // Special sacraficial meat cave where the contacts with no organization go.
      // https://fleetdm.lightning.force.com/lightning/r/Account/0014x000025JC8DAAW/view
      salesforceAccountId = '0014x000025JC8DAAW';
      salesforceAccountOwnerId = '0054x00000735wDAAQ';
    } else {
      let existingAccountRecord = await salesforceConnection.sobject('Account')
      .findOne({
        'Website':  enrichmentData.employer.emailDomain,
        // 'LinkedIn_company_URL__c': enrichmentData.employer.linkedinCompanyPageUrl // TODO: if this information is not present on an existing account, nothing will be returned.
      });
      // console.log(existingAccountRecord);
      if(existingAccountRecord) {
        // Store the ID of the Account record we found.
        salesforceAccountId = existingAccountRecord.Id;
        salesforceAccountOwnerId = existingAccountRecord.OwnerId;
        // console.log('exising account found!', salesforceAccountId);
      } else {


        let roundRobinUsers = await salesforceConnection.sobject('User')
        .find({
          AE_Round_robin__c: true,// eslint-disable-line camelcase
        });
        let userWithEarliestAssignTimeStamp = _.sortBy(roundRobinUsers, 'AE_Account_Assignment_round_robin__c')[0];

        let today = new Date();
        let nowOn = today.toISOString().replace('Z', '+0000');

        salesforceAccountOwnerId = userWithEarliestAssignTimeStamp.Id;

        // Update this user to putthem atthe bottom of the round robin list.
        await salesforceConnection.sobject('User')
        .update({
          Id: salesforceAccountOwnerId,
          // eslint-disable-next-line camelcase
          AE_Account_Assignment_round_robin__c: nowOn
        });
        // If no existing account record was found, create a new one.
        let newAccountRecord = await salesforceConnection.sobject('Account')
        .create({
          OwnerId: salesforceAccountOwnerId,
          Account_Assigned_date__c: nowOn,// eslint-disable-line camelcase
          // eslint-disable-next-line camelcase
          Current_Assignment_Reason__c: 'Inbound Lead',// TODO verify that this matters. if not, do not set it.
          Prospect_Status__c: 'Assigned',// eslint-disable-line camelcase

          Name: enrichmentData.employer.organization,// IFWMIH: We know organization exists
          Website: enrichmentData.employer.emailDomain,
          LinkedIn_company_URL__c: enrichmentData.employer.linkedinCompanyPageUrl,// eslint-disable-line camelcase
          NumberOfEmployees: enrichmentData.employer.numberOfEmployees,
        });
        salesforceAccountId = newAccountRecord.id;
        // console.log('New account created!', salesforceAccountId);
      }
    }



    // Now search for an existing Contact.
    // FUTURE: expand this section to improve the searches.
    let existingContactRecord;
    if(emailAddress){
      // console.log('searching for existing contact by emailAddress');
      existingContactRecord = await salesforceConnection.sobject('Contact')
      .findOne({
        AccountId: salesforceAccountId,
        Email:  emailAddress,
      });
    } else if(linkedinUrl) {
      // console.log('searching for existing contact by linkedInUrl');
      existingContactRecord = await salesforceConnection.sobject('Contact')
      .findOne({
        AccountId: salesforceAccountId,
        LinkedIn_profile__c: linkedinUrl // eslint-disable-line camelcase
      });
    } else {
      existingContactRecord = undefined;
    }

    let salesforceContactId;
    let valuesToSet = {};
    if(emailAddress || enrichmentData.person){
      valuesToSet.Email = emailAddress || enrichmentData.person.emailAddress;
    }
    if(linkedinUrl || enrichmentData.person){
      valuesToSet.LinkedIn_profile__c = linkedinUrl || enrichmentData.person.linkedinUrl;// eslint-disable-line camelcase
    }
    if(enrichmentData.person){
      valuesToSet.Title = enrichmentData.person.title;
    }
    if(primaryBuyingSituation) {
      valuesToSet.Primary_buying_situation__c = primaryBuyingSituation;// eslint-disable-line camelcase
    }
    if(psychologicalStage) {
      valuesToSet.Stage__c = psychologicalStage;// eslint-disable-line camelcase
    }


    if(existingContactRecord){
      salesforceContactId = existingContactRecord.Id;
      // console.log(`existing contact record found! ${salesforceContactId}`);
      // Update the existing contact with the information provided.
      await salesforceConnection.sobject('Contact')
      .update({
        Id: salesforceContactId,
        ...valuesToSet,
      });
      // console.log(`${salesforceContactId} updated!`);
    } else {
      // Otherwise create a new Contact record.
      let newContactRecord = await salesforceConnection.sobject('Contact')
      .create({
        AccountId: salesforceAccountId,
        OwnerId: salesforceAccountOwnerId,
        FirstName: firstName,
        LastName: lastName,
        ...valuesToSet,
      });
      // console.log(newContactRecord);
      salesforceContactId = newContactRecord.id;
      // console.log(`New contact record created! ${salesforceContactId}`);
    }


    return {
      salesforceAccountId,
      salesforceContactId
    };

  }


};

