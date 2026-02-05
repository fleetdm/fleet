module.exports = {


  friendlyName: 'Deliver contact form message',


  description: 'Deliver a contact form message to the appropriate internal channel(s).',


  inputs: {

    emailAddress: {
      required: true,
      isEmail: true,
      type: 'string',
      description: 'A return email address where we can respond.',
      example: 'hermione@hogwarts.edu'
    },

    firstName: {
      required: true,
      type: 'string',
      description: 'The first name of the human sending this message.',
      example: 'Emma'
    },

    lastName: {
      required: true,
      type: 'string',
      description: 'The last name of the human sending this message.',
      example: 'Watson'
    },

    message: {
      type: 'string',
      required: true,
      description: 'The custom message, in plain text.'
    }

  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    },
    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },

  },


  fn: async function({emailAddress, firstName, lastName, message}) {

    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForContactFormSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }

    let userHasPremiumSubscription = false;
    let thisSubscription;
    if(this.req.me){
      thisSubscription = await Subscription.findOne({user: this.req.me.id});
      if(thisSubscription) {
        userHasPremiumSubscription = true;
      }
    }
    let subject = 'New contact form message';
    if(userHasPremiumSubscription) {
      // If the user has a Fleet Premium subscription, prepend the message with details about their subscription.
      let subscriptionDetails =`
Fleet Premium subscription details:
- Fleet Premium subscriber since: ${new Date(thisSubscription.createdAt).toISOString().split('T')[0]}
- Next billing date: ${new Date(thisSubscription.nextBillingAt).toISOString().split('T')[0]}
- Host count: ${thisSubscription.numberOfHosts}
- Organization: ${this.req.me.organization}
-----

      `;
      message = subscriptionDetails + message;
      subject = 'New Fleet Premium customer message';
    }

    // If the submitter has a marketing attribution information set in their cookie, send the details when creating/updating a contact/account/historical record.
    let attributionDetailsOrUndefined = this.req.session.marketingAttribution;// Will be undefined if this is not set.
    // Note: We're using sails.helpers.flow.build INSIDE of a build helper here so that errors from the Salesforce helpers do not prevent the support email from being sent.
    // This is so we can be sure the website has had time to create/update CRM records before unthread attempts creates them with no parent account record.
    sails.helpers.flow.build(async ()=>{

      await sails.helpers.flow.build(async ()=>{
        let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
          emailAddress: emailAddress,
          firstName: firstName,
          lastName: lastName,
          contactSource: 'Website - Contact forms',
          description: `Sent a contact form message: ${message}`,
          marketingAttributionInformation: attributionDetailsOrUndefined
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
          intentSignal: 'Submitted the "Send a message" form',
          eventContent: message,
        }).intercept((err)=>{
          return new Error(`Could not create an historical event. Full error: ${require('util').inspect(err)}`);
        });
      }).tolerate((err)=>{
        sails.log.warn(`When a user submitted a contact form message, a contact/account/historical event could not be created/updated in the CRM for this email address: ${emailAddress}. Full error: ${require('util').inspect(err)}`);
        return;
      });

      await sails.helpers.sendTemplateEmail.with({
        to: sails.config.custom.fromEmailAddress,
        replyTo: {
          name: firstName + ' '+ lastName,
          emailAddress: emailAddress,
        },
        subject,
        layout: false,
        template: 'email-contact-form',
        templateData: {
          emailAddress,
          firstName,
          lastName,
          message,
        },
        ensureAck: true,
      });

    }).exec((err)=>{
      if(err) {
        sails.log.warn(`Background task failed: When a user submitted a contact form message, an error occured when sending an email for their message. Here's the undelivered message:\n Name: ${firstName + ' ' + lastName}, Email: ${emailAddress}, Message: ${message} \nFull error: ${require('util').inspect(err)}`);
      }
      return;
    });//_∏_

  }

};
