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
    success: {
      description: 'The user has opted out of markering emails',
    }
  },


  fn: async function ({emailAddress}) {

    let userRecord = await User.findOne({emailAddress: emailAddress});

    if(!userRecord){
      throw 'userNotFound';
    }
    // Update the user record for this email address to set their nurture email timestamps to 1
    // so they are excluded them from future runs of the deliver-nurture-emails script.
    // FUTURE: update the user model to have a subscribedToNurtureEmails attribute.
    await User.updateOne({emailAddress: emailAddress}).set({
      stageThreeNurtureEmailSentAt: 1,
      stageFourNurtureEmailSentAt: 1,
      stageFiveNurtureEmailSentAt: 1,
    });

    // Update the contact record in salesforce for this email address to indicate that they have opted out of marketing emails.
    if(sails.config.environment === 'production'){
      // Log in to Salesforce.
      let salesforceConnection = await sails.helpers.salesforce.getConnection();

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
          Unsubscribed_from_email_contact__c: true,// eslint-disable-line camelcase
        });
      }
    }
    // Redirect the user to the homepage with a #unsubscribe hash link.
    return this.res.redirect('/#unsubscribed');

  }


};
