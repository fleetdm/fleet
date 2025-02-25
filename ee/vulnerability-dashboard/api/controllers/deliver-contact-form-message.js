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

    if (!sails.config.custom.internalEmailAddress) {
      throw new Error(
`Cannot deliver incoming message from contact form because there is no internal
email address (\`sails.config.custom.internalEmailAddress\`) configured for this
app.  To enable contact form emails, you'll need to add this missing setting to
your custom config -- usually in \`config/custom.js\`, \`config/staging.js\`,
\`config/production.js\`, or via system environment variables.`
      );
    }

    await sails.helpers.sendTemplateEmail.with({
      to: sails.config.custom.internalEmailAddress,
      subject: 'New contact form message',
      template: 'internal/email-contact-form',
      layout: false,
      templateData: {
        contactName: fullName,
        contactEmail: emailAddress,
        topic,
        message,
      }
    });

  }


};
