module.exports = {


  friendlyName: 'Deliver contact form message',


  description: 'Deliver a contact form message to the appropriate internal channel(s).',


  inputs: {

    emailAddress: {
      required: true,
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

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },
    success: {
      description: 'The message was sent successfully.'
    }

  },


  fn: async function({emailAddress, firstName, lastName, message}) {

    if (!sails.config.custom.slackWebhookUrlForContactForm) {
      throw new Error(
        'Message not delivered: slackWebhookUrlForContactForm needs to be configured in sails.config.custom. Here\'s the undelivered message: ' +
        `Name: ${firstName + ' ' + lastName}, Email: ${emailAddress}, Message: ${message ? message : 'No message.'}`
      );
    }

    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }

    await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForContactForm, {
      text: `New contact form message: (Remember: we have to email back; can't just reply to this thread.) cc @sales `+
      `Name: ${firstName + ' ' + lastName}, Email: ${emailAddress}, Message: ${message ? message : 'No message.'}`
    });

    await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress: emailAddress,
      firstName: firstName,
      lastName: lastName,
    });


    // Send a POST request to Zapier
    await sails.helpers.http.post(
      'https://hooks.zapier.com/hooks/catch/3627242/3cxcriz/',
      {
        'emailAddress': emailAddress,
        'firstName': firstName,
        'lastName': lastName,
        'message': message,
        'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret
      }
    )
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed'], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted a contact form message, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}. Raw error: ${err}`);
      return;
    });


  }

};
