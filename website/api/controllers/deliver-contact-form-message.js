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

    await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForContactForm, {
      text: `New contact form message: (cc: <@U0801Q57JDU>) (Remember: we have to email back; can't just reply to this thread.)`+
      `Name: ${firstName + ' ' + lastName}, Email: ${emailAddress}, Message: ${message ? message : 'No message.'}`
    });

    sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress: emailAddress,
      firstName: firstName,
      lastName: lastName,
      contactSource: 'Website - Contact forms',
      description: `Sent a contact form message: ${message}`,
    }).exec((err)=>{// Use .exec() to run the salesforce helpers in the background.
      if(err) {
        sails.log.warn(`Background task failed: When a user submitted a contact form message, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
      }
      return;
    });

  }

};
