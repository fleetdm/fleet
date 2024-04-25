module.exports = {


  friendlyName: 'Update or create contact and account',


  description: 'Upsert contact±account into Salesforce given fresh data about a particular person, and fresh IQ-enrichment data about the person and account.',


  inputs: {

    // Find by…
    emailAddress: { type: 'string', defaultsTo: '' },
    linkedinUrl: { type: 'string', defaultsTo: '' },

    // Set…
    firstName: { type: 'string' },
    lastName: { type: 'string' },
    organization: { type: 'string' },
    primaryBuyingSituation: { type: 'string' },
    psychologicalStage: { type: 'string' },
    numberOfHosts: { type: 'number' },

  },


  exits: {

    success: {
      outputType: {
        salesforceAccountId: 'string',
        salesforceContactId: 'string'
      }
    },

  },


  fn: async function ({emailAddress, linkedinUrl, firstName, lastName, organization, primaryBuyingSituation, psychologicalStage, numberOfHosts}) {
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    require('assert')(sails.config.custom.iqSecret);

    let jsforce = require('jsforce');

    if(!emailAddress && !linkedinUrl){
      throw new Error('UsageError: when updating or creating a contact and account in salesforce, either an email or linkedInUrl is required.');
    }

    let enrichmentData = await sails.helpers.iq.getEnriched(emailAddress, linkedinUrl, firstName, lastName, organization);

    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });

    let salesforceAccountId;
    let salesforceContactId;
    let salesforceAccountOwnerId;// TODO: do we need to build our own round robin, or will this be handled by Salesforce?

    // TODO: wrap this in a try-catch block to handle errors from Salesforce.
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);


    let existingAccountRecord = await salesforceConnection.sobject('Account')
    .findOne({
      'Website':  enrichmentData.employer.emailDomain,
      'LinkedIn_company_URL__c': enrichmentData.employer.linkedinCompanyPageUrl// TODO: if this information is not present on an existing account, nothing will be returned.
    }).execute();


    if(existingAccountRecord !== null){
      // Store the ID of the Account record we found.
      salesforceAccountId = existingAccountRecord.Id;

    } else {
      // If no existing account record was found, create a new one.
      let newAccountRecord = salesforceConnection.sobject('Account')
      .create({
        Name: enrichmentData.employer.organization,
        Website: enrichmentData.employer.emailDomain,
        // eslint-disable-next-line camelcase
        LinkedIn_company_URL__c: enrichmentData.employer.linkedinCompanyPageUrl,
        NumberOfEmployees: enrichmentData.employer.numberOfEmployees,
      }).execute();
      salesforceAccountId = newAccountRecord.Id;
    }

    // Now search for an existing Contact.
    // TODO finding a contact by a linkedIn URL.
    let existingContactRecord = await salesforceConnection.sobject('Contact')
    .findOne({
      'Email':  emailAddress,
      // 'LinkedIn_profile__c': linkedinUrl // TODO: If an existing contact record does not have this value, it will not be returned,
    }).execute();
    // console.log(existingContactRecord);

    if(existingContactRecord !== null){
      salesforceContactId = existingContactRecord.Id;
    } else {
      // Otherwise create a new Contact record.
      let newContactRecord = salesforceConnection.sobject('Contact')
      .create({
        FirstName: firstName,
        LastName: lastName,
        Email: emailAddress,
        Website: enrichmentData.employer.emailDomain,
        // eslint-disable-next-line camelcase
        Primary_buying_situation__c: primaryBuyingSituation,
        // eslint-disable-next-line camelcase
        LinkedIn_profile__c: linkedinUrl,
        // eslint-disable-next-line camelcase
        LinkedIn_company_URL__c: enrichmentData.employer.linkedinCompanyPageUrl,
        NumberOfEmployees: enrichmentData.employer.numberOfEmployees,
        // OwnerId: // TODO (How will we get this?)
        // LeadSource: TODO (Do we need to set this?)
      })
      .execute();
      salesforceContactId = newContactRecord.Id;
    }

    return {
      salesforceAccountId,
      salesforceContactId
    };

  }


};

