module.exports = {


  friendlyName: 'Deliver launch party signup',


  description: 'Delivers a form submission to a Zapier webhook when someone RSVPs to our MDM launch party.',


  inputs: {

    emailAddress: {
      required: true,
      type: 'string',
      description: 'The email address provided when a user submitted the launch party waitlist form.',
      example: 'hermione@hogwarts.edu'
    },

    firstName: {
      required: true,
      type: 'string',
      description: 'The first name provided when a user submitted the launch party waitlist form',
    },

    lastName: {
      required: true,
      type: 'string',
      description: 'The last name provided when a user submitted the launch party waitlist form',
    },

    jobTitle: {
      type: 'string',
      description: 'The job title provided when a user submitted the launch party waitlist form',
    },

    phoneNumber: {
      type: 'string',
      description: 'The phone number provided when a user submitted the launch party waitlist form',
    },
  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    }

  },


  fn: async function({emailAddress, firstName, lastName, jobTitle, phoneNumber}) {

    if(!sails.config.custom.zapierSandboxWebhookSecret) {
      throw new Error('Message not delivered: zapierSandboxWebhookSecret needs to be configured in sails.config.custom.');
    }
    // Send a POST request to Zapier
    await sails.helpers.http.post(
      'https://hooks.zapier.com/hooks/catch/3627242/33kdpw0/',
      {
        'firstName': firstName,
        'lastName': lastName,
        'emailAddress': emailAddress,
        'jobTitle': jobTitle,
        'phoneNumber': phoneNumber,
        'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret
      }
    )
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted the launch party waitlist form, an error occurred while sending a request to Zapier. Raw error: ${require('util').inspect(err)}`);
      return;
    });

  }

};
