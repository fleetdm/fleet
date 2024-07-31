module.exports = {


  friendlyName: 'Deliver deal registration submission',


  description: 'Sends an email with the contents of a deal registration form submission',


  inputs: {
    submittersFirstName: {type: 'string'},
    submittersLastName: {type: 'string'},
    submittersEmailAddress: {type: 'string'},
    submittersOrganization: {type: 'string'},
    customersFirstName: {type: 'string'},
    customersLastName: {type: 'string'},
    customersEmailAddress: {type: 'string'},
    linkedinUrl: {type: 'string', defaultsTo: 'N/A'},
    customersOrganization: {type: 'string'},
    customersCurrentMdm: {type: 'string', defaultsTo: 'N/A'},
    otherMdmEvaluated: {type: 'string', defaultsTo: 'N/A'},
    preferredHosting: {type: 'string', defaultsTo: 'N/A'},
    expectedDealSize: {type: 'string'},
    expectedCloseDate: {type: 'string'},
    notes: {type: 'string', defaultsTo: 'N/A'},
  },


  exits: {

  },


  fn: async function (inputs) {
    if(!sails.config.custom.dealRegistrationContactEmailAddress){
      throw new Error('Missing config variable! Please set sails.config.custom.dealRegistrationContactEmailAddress to be the email address of the person who receives deal registration submissions.');
    }
    // send the information to the deal registration contact email address.
    await sails.helpers.sendTemplateEmail.with({
      to: sails.config.custom.dealRegistrationContactEmailAddress,
      subject: 'New deal registration form submission',
      template: 'email-deal-registration',
      templateData: inputs,
    });
    // All done.
    return;

  }


};
