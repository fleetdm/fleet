module.exports = {


  friendlyName: 'Create quote',


  description: '',


  inputs: {

    // numberOfHosts: {
    //   type: 'number',
    //   required: true,
    // },

    macosHosts: {
      type: 'number',
      defaultsTo: 0,
      // required: true,
    },
    windowsHosts: {
      type: 'number',
      defaultsTo: 0,
      // required: true,
    },
    linuxHosts: {
      type: 'number',
      defaultsTo: 0,
      // required: true,
    },
    iosHosts: {
      type: 'number',
      defaultsTo: 0,
      // required: true,
    },
    androidHosts: {
      type: 'number',
      defaultsTo: 0,
      // required: true,
    },
    otherHosts: {
      type: 'number',
      defaultsTo: 0,
      // required: true,
    },

  },


  exits: {

  },


  fn: async function (inputs) {
    let numberOfHosts = _.sum(_.values(inputs));
    // Determine the price, 7 dollars * host * month (Billed anually)
    let price = 7.00 * numberOfHosts * 12;

    let quote = await Quote.create({
      numberOfHosts: numberOfHosts,
      quotedPrice: price,
      organization: this.req.me.organization,
      user: this.req.me.id,
    }).fetch();


    sails.helpers.flow.build(async ()=>{
      // If the submitter has a marketing attribution cookie, send the details when creating/updating a contact/account/historical record.
      let attributionCookieOrUndefined = this.req.cookies.marketingAttribution;

      let descriptionForContactUpdate =
      `Created a quote for a self-service Fleet Premium License for ${numberOfHosts} hosts.
        - macOS hosts: ${inputs.macosHosts}
        - Windows hosts: ${inputs.windowsHosts}
        - Linux hosts: ${inputs.linuxHosts}
        - iOS/iPadOS hosts: ${inputs.iosHosts}
        - Android hosts: ${inputs.androidHosts}
        - Other hosts: ${inputs.otherHosts}
      `;

      let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: this.req.me.emailAddress,
        firstName: this.req.me.firstName,
        lastName: this.req.me.lastName,
        contactSource: 'Website - Contact forms',
        description: descriptionForContactUpdate,
        marketingAttributionCookie: attributionCookieOrUndefined,
        numberOfHostsDetails: inputs,
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
        intentSignal: 'Created a quote for a self-service Fleet Premium license',
        eventContent: descriptionForContactUpdate,
      }).intercept((err)=>{
        return new Error(`Could not create an historical event. Full error: ${require('util').inspect(err)}`);
      });
    }).exec((err)=>{
      if(err){
        sails.log.warn(`Background task failed: When a user (email: ${this.req.me.emailAddress} created a self-service Fleet premium license quote, a Contact/Account/website activity record could not be created/updated in the CRM.`, require('util').inspect(err));
      }
      return;
    });//_∏_


    return quote;

  }


};
