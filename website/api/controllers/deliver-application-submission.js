module.exports = {


  friendlyName: 'Deliver application submission',


  description: 'Delivers form submissions from the application form on the contact page to a Zapier webhook.',


  inputs: {

    firstName: {
      required: true,
      type: 'string',
      description: 'The first name of the applicant.',
    },

    lastName: {
      required: true,
      type: 'string',
      description: 'The last name of the applicant.',
    },

    emailAddress: {
      required: true,
      isEmail: true,
      type: 'string',
      description: 'A return email address where we can respond.',
    },

    position: {
      type: 'string',
      required: true,
      description: 'The open position this applicant is applying for.'
    },

    linkedinProfileUrl: {
      type: 'string',
      required: true,
      description: 'The URL of the applicant\'s LinkedIn profile.'
    },

    location: {
      type: 'string',
      required: true,
      description: 'The location of the applicant'
    },

    message: {
      type: 'string',
      required: true,
      description: 'The applican\'ts cover letter in plain text.',
    },

  },


  exits: {
    success: {
      description: 'A job application was successfully submitted.',
    }
  },


  fn: async function ({ firstName, lastName, emailAddress, position, linkedinProfileUrl, location, message,}) {




    // Send the submitted information to a Zapier webhook.
    await sails.helpers.http.post.with({
      url: 'https://hooks.zapier.com/hooks/catch/3627242/uwc77dr/',
      data: {
        firstName,
        lastName,
        emailAddress,
        position: position.replace(/🧑‍🚀|🚀|🌦️|🎐|🫧|🐋|🦢|🌐/, ''),// Remove the emoji from the job title.
        linkedinProfileUrl,
        location,
        message,
        webhookSecret: sails.config.custom.zapierWebhookSecret
      }
    })
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted the application form, an error occurred while sending a request to Zapier. Raw error: ${require('util').inspect(err)}`);
      return;
    });


    // All done.
    return;

  }


};
