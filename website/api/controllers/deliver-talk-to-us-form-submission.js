module.exports = {


  friendlyName: 'Deliver talk to us form submission',


  description: '',


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

    organization: {
      type: 'string',
      required: true,
      description: 'The organization of the user who submitted the "talk to us" form'
    },

    numberOfHosts: {
      type: 'string',
      required: true,
      description: 'The organization of the user who submitted the "talk to us" form'
    },

    primaryBuyingSituation: {
      type: 'string',
      required: true,
      description: 'What this user will be using Fleet for',
      isIn: [
        'vm',
        'mdm',
        'eo-it',
        'eo-security',
      ],
    },

  },


  exits: {

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },

  },


  fn: async function ({emailAddress, firstName, lastName, organization, numberOfHosts, primaryBuyingSituation}) {

    const bannedEmailDomainsForContactFormMessages = [
      'gmail.com','yahoo.com', 'yahoo.co.uk','hotmail.com','hotmail.co.uk', 'outlook.com', 'icloud.com', 'proton.me','live.com','yandex.ru','ymail.com',
    ];

    let emailDomain = emailAddress.split('@')[1];

    if(_.includes(bannedEmailDomainsForContactFormMessages, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }
    await sails.helpers.http.post.with({
      url: 'https://hooks.zapier.com/hooks/catch/3627242/3cxwxdo/',
      data: {
        emailAddress,
        firstName,
        lastName,
        organization,
        numberOfHosts,
        primaryBuyingSituation,
        webhookSecret: sails.config.custom.zapierSandboxWebhookSecret,
      }
    })
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed'], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted a contact form message, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}. Raw error: ${err}`);
      return;
    });



    return;

  }


};
