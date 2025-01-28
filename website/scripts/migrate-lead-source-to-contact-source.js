module.exports = {


  friendlyName: 'Migrate lead source to contact source',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run migrate-lead-source-to-contact-source`)');

    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);

    // Log in to Salesforce.
    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    let POSSIBLE_CONTACT_SOURCES = ['Dripify', 'Website - Contact forms', 'Website - Sign up', 'Website - Swag request', 'Manual research', 'Initial qualification meeting'];
    let contacts = (
      await salesforceConnection.query(`SELECT Id, LeadSource, FirstName FROM Contact WHERE Contact_source__c = NULL AND LeadSource IN (${POSSIBLE_CONTACT_SOURCES.map((src)=>'\''+src+'\'').join(', ')})`)
      // await salesforceConnection.query(`SELECT Id, LeadSource, FirstName FROM Contact WHERE LastName = 'McNeil' AND FirstName IN (${['Mike'].map((src)=>'\''+src+'\'').join(', ')}) AND LeadSource IN (${POSSIBLE_CONTACT_SOURCES.map((src)=>'\''+src+'\'').join(', ')})`)
    ).records;// « unpack the sausage
    // console.log(contacts);

    await sails.helpers.flow.simultaneouslyForEach(contacts, async (contact)=>{
      // console.log(`${contact.FirstName} :: ${contact.LeadSource}`);
      await salesforceConnection.sobject('Contact').update({
        Id: contact.Id,
        Contact_source__c: contact.LeadSource//eslint-disable-line camelcase
      });
    });//∞


  }


};

