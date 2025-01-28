module.exports = {

  // TODO: This isn't a thing anymore, we can delete it.
  friendlyName: 'Deliver MDM beta signup',


  description: 'Delivers a message to an internal slack channel to notify us when someone signs up for our live Q&A',


  inputs: {

    emailAddress: {
      required: true,
      type: 'string',
      description: 'The email address provided when a user submitted the MDM beta signup form.',
      example: 'hermione@hogwarts.edu'
    },

    fullName: {
      required: true,
      type: 'string',
      description: 'The name provided when a user submitted the MDM beta signup form',
    },

    jobTitle: {
      required: true,
      type: 'string',
      description: 'The job title provided when a user submitted the MDM beta signup form',
    },

    numberOfHosts: {
      required: true,
      type: 'number',
      description: 'The number of hosts provided when a user submitted the MDM beta signup form',
    },
  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    }

  },


  fn: async function({emailAddress, fullName, jobTitle, numberOfHosts}) {

    if(!sails.config.custom.zapierSandboxWebhookSecret) {
      throw new Error('Message not delivered: zapierSandboxWebhookSecret needs to be configured in sails.config.custom.');
    }

    // Send a POST request to Zapier
    await sails.helpers.http.post(
      'https://hooks.zapier.com/hooks/catch/3627242/bj5nh8y/',
      {
        'emailAddress': emailAddress,
        'fullName': fullName,
        'jobTitle': jobTitle,
        'numberOfHosts': numberOfHosts,
        'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret
      }
    )
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted the MDM beta signup form, an error occurred while sending a request to Zapier. Raw error: ${require('util').inspect(err)}`);
      return;
    });

  }

};
