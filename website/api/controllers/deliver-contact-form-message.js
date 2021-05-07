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

    fullName: {
      required: true,
      type: 'string',
      description: 'The full name of the human sending this message.',
      example: 'Hermione Granger'
    },

    message: {
      required: true,
      type: 'string',
      description: 'The custom message, in plain text.'
    }

  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    }

  },


  fn: async function({emailAddress, topic, fullName, message}) {

    await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForContactForm, {
      text: `New contact form message: (Remember: we have to email back; can't just reply to this thread.) cc @sales `+
      `Name: ${fullName}, Email: ${emailAddress}, Topic: ${topic}, Message: ${message}`
    });

  }


};
