module.exports = {


  friendlyName: 'Unsubscribe from marketing emails',


  description: 'Unsubscribes a specified email address from the nurture email automation.',


  inputs: {
    emailAddress: {
      type: 'string',
      description: 'The email address of the user who wants to unsubscribe from marketing emails.',
      required: true,
    }
  },


  exits: {
    userNotFound: {
      description: 'The provided email address could not be matched to a Fleet user account',
      responseType: 'badRequest',
    },
    userAlreadyUnsubscribed: {
      description: 'The provided email address is already unsubscribed to marketing emails',
      responseType: 'success',
    },
    success: {
      description: 'The user has opted out of markering emails',
    }
  },


  fn: async function ({emailAddress}) {

    let userRecord = await User.findOne({emailAddress: emailAddress});

    if(!userRecord){
      throw 'userNotFound';
    }

    if(!userRecord.subscribedToNurtureEmails){
      throw 'userAlreadyUnsubscribed';
    }

    await User.updateOne({emailAddress: emailAddress}).set({subscribedToNurtureEmails: false});

    if(sails.config.environment === 'production'){
      require('assert')(sails.config.custom.salesforceIntegrationUsername);
      require('assert')(sails.config.custom.salesforceIntegrationPasskey);

      // Log in to Salesforce.
      let jsforce = require('jsforce');
      let salesforceConnection = new jsforce.Connection({
        loginUrl : 'https://fleetdm.my.salesforce.com'
      });
      await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

      let existingContactRecord = await salesforceConnection.sobject('Contact')
      .findOne({
        Email:  emailAddress,
      });

      if(existingContactRecord) {
        //If we found an existing contact record in salesforce, update its status to be "Do not contact"
        let salesforceContactId = existingContactRecord.Id;
        await salesforceConnection.sobject('Contact')
        .update({
          Id: salesforceContactId,
          Stage__c: 'Do not contact',// eslint-disable-line camelcase
        });
      }
    }
    // All done.
    return;

  }


};
