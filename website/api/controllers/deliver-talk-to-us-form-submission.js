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



    // Simultaneously run the enrichment helper and send a prompt to an LLM to try to guess the company's headquarters location (city/country/state) from the email domain.
    // We use the information returned by the LLM as a fallback for the territory lookup below when the enrichment helper doesn't return location information.
    let locationGuessSystemPrompt = 'You are a precise data-extraction function. Respond with a single raw JSON object and nothing else.';
    let locationGuessPrompt =
`Where is the company that owns the email domain "${emailDomain}" headquartered?

Respond with a JSON object using these keys:
- "city": the headquarters city name.
- "country": the full country name in English (for example, "United States").
- "state": the full state name. Only include this key when the country is the United States.

Only include a key when you are confident of its value. Omit any key you are unsure of; do not guess, and do not use null or empty strings.`;

    let { enrichmentInformation, locationGuessFromEmailDomain } = await sails.helpers.flow.simultaneously({
      enrichmentInformation: async()=>{
        return await sails.helpers.iq.getEnriched.with({
          emailAddress,
          includeEmployerHeadquartersInformation: true,
        }).tolerate((err)=>{
          sails.log.warn(`When a user (${emailAddress}) submitted the "Talk to us form", an error occurred while getting enrichment information for this user. Error from get-enriched helper: ${require('util').inspect(err)}`);
          return {};
        });
      },
      locationGuessFromEmailDomain: async()=>{
        return await sails.helpers.ai.prompt.with({
          prompt: locationGuessPrompt,
          baseModel: 'gpt-5-nano-2025-08-07',
          expectJson: true,
          systemPrompt: locationGuessSystemPrompt,
        }).tolerate((err)=>{
          sails.log.warn(`When a user (${emailAddress}) submitted the "Talk to us form", an error occurred while guessing their company's headquarters location from the email domain (${emailDomain}). Error from prompt helper: ${require('util').inspect(err)}`);
          return {};
        });
      },
    });

    let employeeCountFromEnrichmentHelper = (enrichmentInformation.employer || {}).numberOfEmployees;

    // If we got a employer.numberOfEmployees value from the getEnriched helper, or the user entered more than 700 hosts, get the SF user who owns the territory that this user's company is in, and send them to a "Talk to us" calendly event.
    if(numberOfHosts >= 700 || (employeeCountFromEnrichmentHelper && employeeCountFromEnrichmentHelper >= 700)) {
      contactInformation.contactSource = 'Website - Contact forms - Demo - ICP';

      if(employeeCountFromEnrichmentHelper && employeeCountFromEnrichmentHelper >= 700) {
        contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Talk to us" event because of the number of employees (${employeeCountFromEnrichmentHelper}) returned by Coresignal. Provided organization name: ${organization}, Number of employees: ${numberOfHosts}`;
      } else {
        contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Talk to us" event. Provided organization name: ${organization}, Number of employees: ${numberOfHosts}`;
      }

      sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
        if(err) {
          sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}. Full Error: ${require('util').inspect(err)}`);
        }
      });//_∏_
      // Prefer the enrichment helper's location, but fall back to the email-domain location
      // guess when enrichment didn't return a country (the field getTerritoryUserId requires).
      let employerLocation = enrichmentInformation.employer || {};
      let locationForTerritoryLookup = (employerLocation.country ? employerLocation : locationGuessFromEmailDomain) || {};// Default to an empty object if neither sources returned this information.

      let territoryUserId = await sails.helpers.salesforce.getTerritoryUserId.with({
        state: locationForTerritoryLookup.state,
        country: locationForTerritoryLookup.country,
        city: locationForTerritoryLookup.city
      }).tolerate((err)=>{
        sails.log.warn(`When a user submitted the "Talk to us" form, Salesforce territory information could not be found using the provided information. This user will be sent to the calendly link for the washingtonDc region. Full error: ${require('util').inspect(err)}`);
        return '0054x0000086sOlAAI';
      });

      let bookingUrlByUserId = {
        '005UG000006YYDVYA4': 'https://calendly.com/d/d3fs-28g-vdk/talk-to-us', //newYorkCity
        '0054x0000086sOlAAI': 'https://calendly.com/d/dzyz-tt7-yt8/talk-to-us', //washingtonDc
        '005UG000008y0wbYAA': 'https://calendly.com/d/ds9c-9vt-mz6/talk-to-us', //losAngeles
        '0054x0000086wsGAAQ': 'https://calendly.com/d/dz4c-mjx-6xv/talk-to-us', //sanFrancisco
        '005UG000009NnSfYAK': 'https://calendly.com/d/ds88-n2m-ddt/talk-to-us', //stockholm
      };

      let eventUrlForThisUsersTerritory = bookingUrlByUserId[territoryUserId];
      if(!eventUrlForThisUsersTerritory) {
        // If the user ID returned by the helper is not one of the five expected values above, log a warning to alert us, and send the user to the washingtonDc calednly link.
        sails.log.warn(`When looking up Salesforce territory information to route a user (email: ${emailAddress}) who submitted the "Talk to us" form to the correct meeting link, the user ID returned by the getTerritoryUserId helper (${territoryUserId}) did not match the hardcoded user IDs in the bookingUrlByUserId dictionary. This user will be sent to the callendly link for the washingtonDc region.`);
        eventUrlForThisUsersTerritory = 'https://calendly.com/d/dzyz-tt7-yt8/talk-to-us';
      }
      return {
        icp: true,
        eventUrl: eventUrlForThisUsersTerritory +`?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`,
      };
    } else {
      // If the enrichment helper didn't return a employer.numberOfEmployees value and this user has <700 hosts, send them to the "Let's get you set up!" Calendly event
      contactInformation.contactSource = 'Website - Contact forms - Demo';
      contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Let\'s get you set up!" event. Provided organization name: ${organization}, Number of employees: ${numberOfHosts}`;
      sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
        if(err) {
          sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
        }
      });//_∏_
      return {
        icp: false,
        eventUrl: `https://calendly.com/fleetdm/chat?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`
      };
    }
  }


};
