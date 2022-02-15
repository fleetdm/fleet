module.exports = {


  friendlyName: 'Deliver demo signup',


  description: 'Delivers a message to an internal slack channel to notify us when someone signs up for our live Q&A',


  inputs: {

    emailAddress: {
      required: true,
      type: 'string',
      description: 'A return email address where we can respond.',
      example: 'hermione@hogwarts.edu'
    }
  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    }

  },


  fn: async function({emailAddress}) {

    if (sails.config.environment === 'production') {
      if (!sails.config.custom.slackWebhookUrlForContactForm) {
        throw new Error(
          'Message not delivered: slackWebhookUrlForContactForm needs to be configured in sails.config.custom. Here\'s the undelivered message: ' +
          `Email: ${emailAddress}`
        );
      } else {
        await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForContactForm, {
          text: `New demo session signup: (Remember: we have to invite them to the next demo session.) cc @sales`+
          `Email: ${emailAddress}`
        });
      }
    } else {
      sails.log(
        `New demo session signup: (Remember: we have to invite them to the next demo session.) cc @sales`+
        `Email: ${emailAddress}`
      );
    }
  }

};
