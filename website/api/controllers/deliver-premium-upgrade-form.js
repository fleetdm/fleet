module.exports = {


  friendlyName: 'Deliver premium upgrade form',


  description: 'Delivers a Fleet Premium upgrade form submission to a Zapier webhook',


  inputs: {
    organization: {
      type: 'string',
      required: true,
    },

    monthsUsingFleetFree: {
      type: 'string',
      required: true,
      example: '1 - 3 months'
    },

    emailAddress: {
      type: 'string',
      isEmail: true,
      required: true,
    },

    numberOfHosts: {
      type: 'number',
      required: true,
      isInteger: true,
    }
  },


  exits: {
    success: {
      description: 'The Fleet Premium upgrade form submission was sent to Zapier successfully.'
    }
  },


  fn: async function ({organization, monthsUsingFleetFree, emailAddress, numberOfHosts}) {

    if(!sails.config.custom.zapierSandboxWebhookSecret) {
      throw new Error('Message not delivered: zapierSandboxWebhookSecret needs to be configured in sails.config.custom.');
    }

    // Send a POST request to Zapier
    await sails.helpers.http.post(
      'https://hooks.zapier.com/hooks/catch/3627242/bvxxkjf/',
      {
        'emailAddress': emailAddress,
        'organization': organization,
        'numberOfHosts': numberOfHosts,
        'monthsUsingFleetFree': monthsUsingFleetFree,
        'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret
      }
    )
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted the Fleet Premium upgrade form, an error occurred while sending a request to Zapier. Raw error: ${require('util').inspect(err)}`);
      return;
    });//∞

    // All done.
    return;
  }


};
