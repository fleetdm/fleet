module.exports = {


  friendlyName: 'Get enriched',


  description: 'Search for the contact indicated and return enriched data.',


  extendedDescription: `Note about coresignal.com from their FAQ:
    Q: Do you have emails or phone numbers in your database?
    A: No, we don't have emails or phone numbers. We only hold publicly available data on companies and professionals.`,


  moreInfoUrl: 'https://coresignal.com/faq/',


  inputs: {

    emailAddress: { type: 'string', defaultsTo: '', },
    linkedinUrl: { type: 'string', defaultsTo: '', },
    firstName: { type: 'string', defaultsTo: '', },
    lastName: { type: 'string', defaultsTo: '', },
    organization: { type: 'string', defaultsTo: '', },

  },


  exits: {

    success: {
      outputFriendlyName: 'Report',
      outputDescription: 'All available, enriched info about this person and their current employer.',
      outputType: {
        person: {
          linkedinUrl: 'string',
          firstName: 'string',
          lastName: 'string',
          organization: 'string',
          title: 'string',
        },
        employer: {
          organization: 'string',
          numberOfEmployees: 'number',
          emailDomain: 'string',
          linkedinCompanyPageUrl: 'string',
        }
      }
    },

  },


  fn: async function ({emailAddress,linkedinUrl,firstName,lastName,organization}) {

    require('assert')(sails.config.custom.iqSecret);// FUTURE: Rename this config
    require('assert')(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS);

    sails.log.verbose('Enriching from…', emailAddress,linkedinUrl,firstName,lastName,organization);

    // Gather initial information that is obtainable just from parsing provided inputs.
    let emailDomain;
    if (emailAddress) {
      let matches = emailAddress.match(/@([^@]+)$/);
      if (Array.isArray(matches)) {
        emailDomain = matches[1] || undefined;
      }
    }//ﬁ

    let linkedinPersonIdOrUrlSlug;
    if (linkedinUrl) {
      let matches = linkedinUrl.match(/linkedin\.com\/in\/([^/]+)\/?$/);
      if (Array.isArray(matches)) {
        linkedinPersonIdOrUrlSlug = matches[1] || undefined;
      }
    }//ﬁ


    // If no linkedin URL was provided for the person, then also do a website+name+orgName search
    // vs contacts to try and locate the person's linkedin URL.
    //
    // [?] Why?  It provides us with a better unique id than an email.  For example, consider
    //     how everyone has more than one email.  This way, we can avoid sending any emails that
    //     people might experience as "spam", even if they unsubscribe from a different email.
    if (!linkedinPersonIdOrUrlSlug && (firstName || lastName || emailAddress)) {
      let searchBy = {};
      if (firstName && !lastName) {
        searchBy.name = firstName;
      } else if (!firstName && lastName) {
        searchBy.name = lastName;
      } else if (firstName && lastName) {
        searchBy.name = firstName + ' ' + lastName;
      } else {
        searchBy.name = _.startCase(emailAddress.replace(/@[^@]+$/,'').replace(/\./g,' ').replace(/[0-9\-]/g,''));
      }
      if (emailDomain) {
        searchBy.experience_company_website_url = emailDomain;//eslint-disable-line camelcase
        searchBy.active_experience = true;//eslint-disable-line camelcase
      }//ﬁ
      if (organization) {
        searchBy.experience_company_name = organization;//eslint-disable-line camelcase
        searchBy.active_experience = true;//eslint-disable-line camelcase
      }//ﬁ
      if (Object.keys(searchBy).length >= 1) {
        // [?] https://dashboard.coresignal.com/get-started
        let matchingLinkedinPersonIds = await sails.helpers.http.post('https://api.coresignal.com/cdapi/v1/linkedin/member/search/filter', searchBy, {
          Authorization: `Bearer ${sails.config.custom.iqSecret}`,
          'content-type': 'application/json'
        }).tolerate((err)=>{
          sails.log.info(`Failed to enrich (${emailAddress},${linkedinUrl},${firstName},${lastName},${organization}):`,err);
          return [];
        });
        linkedinPersonIdOrUrlSlug = matchingLinkedinPersonIds[0];
      }//ﬁ
    }//ﬁ

    let person;
    let matchingLinkedinCompanyPageId;

    if (linkedinPersonIdOrUrlSlug) {
      // [?] https://dashboard.coresignal.com/get-started
      let matchingPersonInfo = await sails.helpers.http.get('https://api.coresignal.com/cdapi/v1/linkedin/member/collect/'+encodeURIComponent(linkedinPersonIdOrUrlSlug), {}, {
        Authorization: `Bearer ${sails.config.custom.iqSecret}`,
        'content-type': 'application/json'
      }).tolerate((err)=>{
        sails.log.info(`Failed to enrich (${emailAddress},${linkedinUrl},${firstName},${lastName},${organization}):`,err);
        return undefined;
      });

      if (matchingPersonInfo) {

        require('assert')(Array.isArray(matchingPersonInfo.member_experience_collection));
        let matchingWorkExperience;
        if(organization){
          // If organization was provided, we know it is listed in this person's work experience so we'll use it to filter the results.
          matchingWorkExperience = (
            matchingPersonInfo.member_experience_collection.filter((workExperience) =>
              !workExperience.deleted &&
              !workExperience.date_to &&
              workExperience.company_name === organization
            )
          )[0];
        } else {
          // Otherwise, we'll use the top experience on this user's profile.
          matchingWorkExperience = (
            matchingPersonInfo.member_experience_collection.filter((workExperience) =>
              !workExperience.deleted &&
              workExperience.order_in_profile === 1 &&
              !workExperience.date_to
            )
          )[0];
        }//ﬁ

        let matchedOrganizationName;
        let matchedTitle;
        if (matchingWorkExperience) {
          matchedOrganizationName = matchingWorkExperience.company_name;
          matchedTitle = matchingWorkExperience.title;
          matchingLinkedinCompanyPageId = matchingWorkExperience.company_id;// « save for use below
        }

        person = {
          linkedinUrl: matchingPersonInfo.canonical_url.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS,''),
          firstName: matchingPersonInfo.first_name,
          lastName: matchingPersonInfo.last_name,
          organization: matchedOrganizationName || '',
          title: matchedTitle || ''
        };

        if (linkedinUrl && person.linkedinUrl && person.linkedinUrl !== linkedinUrl) {
          sails.log.info(`Unexpected result when enriching: Matched linkedin URL for person (${person.linkedinUrl}) does not equal the provided linkedin URL (${linkedinUrl})`);
        }//ﬁ
        if (firstName && person.firstName && person.firstName !== firstName) {
          sails.log.info(`Unexpected result when enriching: Matched current firstName for person (${person.firstName}) does not equal the provided "firstName" (${firstName})`);
        }//ﬁ
        if (lastName && person.lastName && person.lastName !== lastName) {
          sails.log.info(`Unexpected result when enriching: Matched current lastName for person (${person.lastName}) does not equal the provided "lastName" (${lastName})`);
        }//ﬁ
        if (organization && person.organization && person.organization !== organization) {
          sails.log.info(`Unexpected result when enriching: Matched current TOP organization for person (${person.organization}) does not equal the provided "organization" (${organization})`);
        }//ﬁ
      }//ﬁ
    }//ﬁ




    // Now look up the employer.
    //
    // [?] Either use the matched linkedin company page ID from above,
    //     or if no match, then try to find the linkedin company page ID
    //     by other means.  If nothing works, then give up and don't enrich.
    if (!matchingLinkedinCompanyPageId) {
      let searchBy = {};
      if (emailDomain) {
        searchBy.website = emailDomain;
      }//ﬁ
      if (organization) {
        searchBy.name = organization;
      }//ﬁ
      if (Object.keys(searchBy).length >= 1) {
        // [?] https://dashboard.coresignal.com/get-started
        let matchingLinkedinCompanyPageIds = await sails.helpers.http.post('https://api.coresignal.com/cdapi/v1/linkedin/company/search/filter', searchBy, {
          Authorization: `Bearer ${sails.config.custom.iqSecret}`,
          'content-type': 'application/json'
        }).tolerate((err)=>{
          sails.log.info(`Failed to enrich (${emailAddress},${linkedinUrl},${firstName},${lastName},${organization}):`,err);
          return [];
        });

        // If name and domain were used for searching the org, yet no matches found,
        // try searching again, but this time w/o the org name.
        if (matchingLinkedinCompanyPageIds.length === 0 && searchBy.name && searchBy.website) {
          delete searchBy.name;
          // [?] https://dashboard.coresignal.com/get-started
          matchingLinkedinCompanyPageIds = await sails.helpers.http.post('https://api.coresignal.com/cdapi/v1/linkedin/company/search/filter', searchBy, {
            Authorization: `Bearer ${sails.config.custom.iqSecret}`,
            'content-type': 'application/json'
          }).tolerate((err)=>{
            sails.log.info(`Failed to enrich (${emailAddress},${linkedinUrl},${firstName},${lastName},${organization}):`,err);
            return [];
          });
        }//ﬁ

        matchingLinkedinCompanyPageId = matchingLinkedinCompanyPageIds[0];
      }//ﬁ
    }//ﬁ

    let employer;
    if (matchingLinkedinCompanyPageId) {
      // [?] https://dashboard.coresignal.com/get-started
      let matchingCompanyPageInfo = await sails.helpers.http.get('https://api.coresignal.com/cdapi/v1/linkedin/company/collect/'+encodeURIComponent(matchingLinkedinCompanyPageId), {}, {
        Authorization: `Bearer ${sails.config.custom.iqSecret}`,
        'content-type': 'application/json'
      }).tolerate((err)=>{
        sails.log.info(`Failed to enrich (${emailAddress},${linkedinUrl},${firstName},${lastName},${organization}):`,err);
        return undefined;
      });
      if (matchingCompanyPageInfo) {
        let parsedCompanyEmailDomain = require('url').parse(matchingCompanyPageInfo.website);
        // If a company's website does not include the protocol (https://), url.parse will return null as the hostname, if this happens, we'll use the href value returned instead.
        let emailDomain = parsedCompanyEmailDomain.hostname ? parsedCompanyEmailDomain.hostname.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS,'') : parsedCompanyEmailDomain.href.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS,'');
        employer = {
          organization: matchingCompanyPageInfo.name,
          numberOfEmployees: matchingCompanyPageInfo.employees_count,
          emailDomain: emailDomain,
          linkedinCompanyPageUrl: matchingCompanyPageInfo.canonical_url.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS,''),
        };
        if (organization && employer.organization && employer.organization !== organization) {
          sails.log.info(`Unexpected result when enriching: Matched organization name (${employer.organization}) does not equal the provided "organization" (${organization})`);
        }//ﬁ
        if (emailDomain && employer.emailDomain && employer.emailDomain !== emailDomain) {
          sails.log.info(`Unexpected result when enriching: Email domain inferred from matched organization website (${employer.emailDomain}) does not equal the parsed email domain (${emailDomain}) that was derived from the provided "emailAddress" (${emailAddress})`);
        }//ﬁ

        // Use OpenAI to try and enrich some additional data, if it's missing.
        if (!employer.numberOfEmployees) {
          if (!sails.config.custom.openAiSecret) {
            throw new Error('sails.config.custom.openAiSecret not set.');
          }//•

          let prompt = `How many employees does the organization who owns ${emailDomain} have?
    
    Please respond in this form (but instead of 0, put the number of employees, as an integer:
    {
      "employees": 0
    }`;
          let BASE_MODEL = 'gpt-4o';// The base model to use.  https://platform.openai.com/docs/models/gpt-4
          // [?] API: https://platform.openai.com/docs/api-reference/chat/create
          let openAiResponse = await sails.helpers.http.post('https://api.openai.com/v1/chat/completions', {
            model: BASE_MODEL,
            messages: [ { role: 'user', content: prompt } ],// // https://platform.openai.com/docs/guides/chat/introduction
            temperature: 0.7,
            max_tokens: 256//eslint-disable-line camelcase
          }, {
            Authorization: `Bearer ${sails.config.custom.openAiSecret}`
          })
          .tolerate((unusedErr)=>{});

          if (openAiResponse) {
            try {
              employer.numberOfEmployees = JSON.parse(openAiResponse.choices[0].message.content).employees;
            } catch (unusedErr) {
              employer.numberOfEmployees = 1;
            }
          }//ﬁ
        }//ﬁ
      }//ﬁ
    }//ﬁ

    return {
      person,
      employer
    };

  }

};
