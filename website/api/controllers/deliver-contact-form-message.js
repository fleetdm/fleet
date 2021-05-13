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

    topic: {
      required: true,
      type: 'string',
      description: 'The topic from the contact form.',
      example: 'I want to buy stuff.'
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
      required: false,
      type: 'string',
      description: 'The custom message, in plain text.'
    }

  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    }

  },


  fn: async function({emailAddress, topic, firstName, lastName, message}) {

    if (!sails.config.custom.slackWebhookUrlForContactForm) {
      throw new Error(
        'Message not delivered: slackWebhookUrlForContactForm needs to be configured in sails.config.custom. Here\'s the undelivered message: ' +
        `Name: ${firstName + ' ' + lastName}, Email: ${emailAddress}, Topic: ${topic}, Message: ${message ? message : 'No message.'}`
      );
    } else {
      await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForContactForm, {
        text: `New contact form message: (Remember: we have to email back; can't just reply to this thread.) cc @sales `+
        `Name: ${firstName + ' ' + lastName}, Email: ${emailAddress}, Topic: ${topic}, Message: ${message ? message : 'No message.'}`
      });
    }
  }

};
