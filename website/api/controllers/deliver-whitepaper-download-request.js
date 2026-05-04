module.exports = {


  friendlyName: 'Deliver whitepaper download request',


  description: '',


  inputs: {
    firstName: {type: 'string', required: true },
    lastName: {type: 'string', required: true },
    emailAddress: {type: 'string', required: true, isEmail: true },
    whitepaperName: {type: 'string', required: true },
  },


  exits: {
    success: {description: 'A users whitepaper download request was successfully submitted.'},
    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },

  },


  fn: async function ({firstName, lastName, emailAddress, whitepaperName}) {

    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }

    // If the submitter has a marketing attribution cookie, send the details when creating/updating a contact/account/historical record.
    let attributionCookieOrUndefined = this.req.cookies.marketingAttribution;

    sails.helpers.flow.build(async ()=>{
      let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: emailAddress,
        firstName: firstName,
        lastName: lastName,
        contactSource: 'Website - Gated document',
        description: `Submitted a form to download the ${whitepaperName} whitepaper.`,
        marketingAttributionCookie: attributionCookieOrUndefined
      }).intercept((err)=>{
        return new Error(`Could not create/update a contact or account. Full error: ${require('util').inspect(err)}`);
      });

      // If the Contact record returned by the updateOrCreateContactAndAccount does not have a parent Account record, throw an error to stop the build helper.
      if(!recordIds.salesforceAccountId) {
        throw new Error(`Could not create historical event. The contact record (ID: ${recordIds.salesforceContactId}) returned by the updateOrCreateContactAndAccount helper is missing a parent account record.`);
      }
      // Create the new Fleet website page view record.
      await sails.helpers.salesforce.createHistoricalEvent.with({
        salesforceAccountId: recordIds.salesforceAccountId,
        salesforceContactId: recordIds.salesforceContactId,
        eventType: 'Intent signal',
        intentSignal: 'Requested whitepaper download',
        eventContent: whitepaperName,
      }).intercept((err)=>{
        return new Error(`Could not create an historical event. Full error: ${require('util').inspect(err)}`);
      });
    }).exec((err)=>{
      if(err){
        sails.log.warn(`Background task failed: When a user (email: ${emailAddress} submitted a form to download the ${whitepaperName} whitepaper, a Contact/Account/website activity record could not be created/updated in the CRM.`, require('util').inspect(err));
      }
      return;
    });//_∏_

    // All done.
    return;

  }


};
