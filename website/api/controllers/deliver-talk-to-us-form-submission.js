module.exports = {


  friendlyName: 'Deliver talk to us form submission',


  description: '',


  inputs: {
    emailAddress: {
      required: true,
      isEmail: true,
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
        'it-major-mdm',
        'it-gap-filler-mdm',
        'it-misc',
        'security-misc',
        'security-vm',
      ],
    },

  },


  exits: {

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },
    success: {
      decription: 'A user successfully submitted the "Talk to us" form.',
      outputType: {
        icp: 'boolean',
        eventUrl: 'string',
      },
    }

  },


  fn: async function ({emailAddress, firstName, lastName, organization, numberOfHosts, primaryBuyingSituation}) {

    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }
    let attributionCookieOrUndefined = this.req.cookies.marketingAttribution;

    let contactInformation = {
      emailAddress: emailAddress,
      firstName: firstName,
      lastName: lastName,
      // organization: organization, // Note: the user-provided organization is not used here because we're relying on the enrichment helper below to find the correct organization for this person.
      primaryBuyingSituation: primaryBuyingSituation === 'security-misc' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'it-misc' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'it-major-mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'it-gap-filler-mdm' ? 'IT - Gap-filler MDM' : primaryBuyingSituation === 'security-vm' ? 'Vulnerability management' : undefined,
      psychologicalStage: '4 - Has use case',
      psychologicalStageChangeReason: 'Website - Contact forms',
      marketingAttributionCookie: attributionCookieOrUndefined
    };



    // TODO: Send prompt to try to find this information in paralell.
    let enrichmentInformation = await sails.helpers.iq.getEnriched.with({
      emailAddress,
    }).tolerate((err)=>{
      sails.log.warn(`When a user (${emailAddress}) submitted the "Talk to us form", an error occured while getting enrichment information for this user. Error from get-enriched helper: ${require('util').inspect(err)}`);
      return {};
    });

    // If we got a employer.numberOfEmployees value from the getEnriched helper, send the user to the "talk to us" calendly event if it is 700+.
    if(enrichmentInformation.employer && enrichmentInformation.employer.numberOfEmployees && enrichmentInformation.employer.numberOfEmployees >= 700) {
      contactInformation.contactSource = 'Website - Contact forms - Demo - ICP';
      contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Talk to us" event because of the number of employees (${enrichmentInformation.employer.numberOfEmployees}) returned by Coresignal. Provided organization name: ${organization}, Number of employees: ${numberOfHosts}`;
      sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
        if(err) {
          sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}. Full Error: ${require('util').inspect(err)}`);
        }
      });
      console.log(enrichmentInformation);
      let teritoryUserId = await sails.helpers.salesforce.getTerritoryUserId.with({
        state: enrichmentInformation.employer.state,
        country: enrichmentInformation.employer.country,
        city: enrichmentInformation.employer.city
      }).tolerate((err)=>{
        sails.log.warn(`When a user submitted the "Talk to us" form, Salesforce teritory information could not be found using the provided information. This user will be sent to the calendly link for the washingtonDc region. Full error: ${require('util').inspect(err)}`)
        return '0054x0000086sOlAAI';
      });
      let bookingUrlByUserId = {
        '005UG000006YYDVYA4': 'https://calendly.com/d/d3fs-28g-vdk/talk-to-us', //newYorkCity
        '0054x0000086sOlAAI': 'https://calendly.com/d/dzyz-tt7-yt8/talk-to-us', //washingtonDc
        '005UG000008y0wbYAA': 'https://calendly.com/d/ds9c-9vt-mz6/talk-to-us', //losAngeles
        '0054x0000086wsGAAQ': 'https://calendly.com/d/dz4c-mjx-6xv/talk-to-us', //sanFrancisco
        '005UG000009NnSfYAK': 'https://calendly.com/d/ds88-n2m-ddt/talk-to-us', //stockholm
      };

      let eventUrlForThisUsersTeritory = bookingUrlByUserId[teritoryUserId];

      return {
        icp: true,
        eventUrl: eventUrlForThisUsersTeritory,
      };
    } else {
      // If the enrichment helper didn't return a employer.numberOfEmployees value and this user has <700 hosts, send them to the "Let's get you set up!" Calendly event
      contactInformation.contactSource = 'Website - Contact forms - Demo';
      contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Let\'s get you set up!" event. Provided organization name: ${organization}, Number of employees: ${numberOfHosts}`;
      sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
        if(err) {
          sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
        }
      });
      return {
        icp: false,
        eventUrl: `https://calendly.com/fleetdm/chat?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`
      };
    }
  }


};
