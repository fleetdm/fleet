module.exports = {


  friendlyName: 'Get enriched',


  description: 'Search for the contact indicated and return enriched data.',


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
          emailAddress: 'string',
          linkedinUrl: 'string',
          firstName: 'string',
          lastName: 'string',
          organization: 'string',
          title: 'string',
          phone: 'string',
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

    // curl -X POST "https://api.coresignal.com/cdapi/v1/linkedin/company/search/filter"
    // -H "accept: application/json"
    // -H "Authorization: Bearer {JWT}"
    // -H "Content-Type: application/json"
    // -d "{\"location\":\"United States\",\"industry\":\"Information Technology\",
    //  \"last_updated_gte\":\"2022-05-01 00:00:00\"}"

    // console.log(emailAddress,linkedinUrl,firstName,lastName,organization);

    sails.log.verbose('ignoring linkedinUrl, firstName, and lastName for now...', linkedinUrl,firstName,lastName);

    let emailDomain = '';
    if (emailAddress) {
      emailDomain = emailAddress.match(/@([^@]+)$/) && emailAddress.match(/@([^@]+)$/)[1] || '';
    }//ﬁ

    // [?] https://dashboard.coresignal.com/get-started
    let searchBy = {};
    if (emailAddress) {
      searchBy.website = emailDomain;
    }
    if (organization) {
      searchBy.name = organization;
    }
    let matchingIds = await sails.helpers.http.post('https://api.coresignal.com/cdapi/v1/linkedin/company/search/filter', searchBy, {
      Authorization: `Bearer ${sails.config.custom.iqSecret}`,
      'content-type': 'application/json'
    });
    // console.log('matches:',matchingIds);
    let matchingId = matchingIds[0];
    if (!matchingId) {
      return {
        person: undefined,
        employer: undefined
      };
    }//•  (TODO: replace this temporary hack with something nicer, just prioritizing the important part)
    require('assert')(matchingId);

    let matchingOrgRecord = await sails.helpers.http.get('https://api.coresignal.com/cdapi/v1/linkedin/company/collect/'+encodeURIComponent(matchingId), {}, {
      Authorization: `Bearer ${sails.config.custom.iqSecret}`,
      'content-type': 'application/json'
    });

    // console.log(report);
    return {
      // TODO: the rest
      employer: {
        name: matchingOrgRecord.name,
        numberOfEmployees: matchingOrgRecord.employees_count,
        emailDomain: emailDomain,
        linkedinCompanyPageUrl: matchingOrgRecord.canonical_url,
      }
    };



    // require('assert')(sails.config.custom.iqSecret);

    // let RX_TECHNOLOGY_CATEGORIES = /(device|security|endpoint|configuration management|data management platforms|mobility management|identity|information technology|IT$|employee experience|apple)/i;

    // // [?] https://developer.leadiq.com/#query-searchPeople
    // // [?] https://developer.leadiq.com/#definition-SearchPeopleInput
    // // [?] https://graphql.org/learn/serving-over-http/

    // let searchExpr = `{
    //   ${emailAddress? 'email: '+ JSON.stringify(emailAddress) : ''}
    //   ${linkedinUrl? 'linkedinUrl: '+ JSON.stringify(linkedinUrl) : ''}
    //   ${firstName? 'firstName: '+ JSON.stringify(firstName) : ''}
    //   ${lastName? 'lastName: '+ JSON.stringify(lastName) : ''}
    //   ${organization || emailDomain ? (`company: {
    //     ${organization? 'name: '+ JSON.stringify(organization) : ''}
    //     ${emailDomain? 'domain: '+ JSON.stringify(emailDomain)+' '+'emailDomain: '+ JSON.stringify(emailDomain) : ''}
    //     searchInPastCompanies: false
    //     strict: false
    //   }`) : ''}
    // }`; //sails.log('GraphQL query:',searchExpr);
    // let report = await sails.helpers.http.get('https://api.leadiq.com/graphql', {
    //   query: `{ searchPeople(input: ${searchExpr}) {
    //       totalResults
    //       results {
    //         _id
    //         name { first last }
    //         linkedin { linkedinId linkedinUrl status updatedAt }
    //         profiles { network id username url status updatedAt }
    //         location { country areaLevel1 city fullAddress type status updatedAt }
    //         personalPhones { value type status verificationStatus }
    //         currentPositions {
    //           title
    //           emails { value type status }
    //           phones { value type status verificationStatus }
    //           companyInfo {
    //             name
    //             domain
    //             country
    //             address
    //             linkedinUrl
    //             numberOfEmployees
    //             technologies { name category parentCategory attributes categories }
    //           }
    //         }
    //       }
    //     }
    //   }`,
    // }, {
    //   Authorization: `Basic ${sails.config.custom.iqSecret}`,
    //   'content-type': 'application/json'
    // });

    // if (report.errors) {
    //   sails.log.warn('Errors returned from IQ API when attempting to search for a matching contact:',report.errors);
    // }

    // // sails.log('person search results:',require('util').inspect(report.data.searchPeople.results, {depth:null}));
    // let foundPerson = report.data.searchPeople.results[0]; //sails.log('Found person:',foundPerson);
    // let foundPosition = foundPerson && foundPerson.currentPositions && foundPerson.currentPositions.length >= 1 ? foundPerson.currentPositions[0] : undefined;

    // let person;
    // if (foundPerson) {
    //   person = {
    //     emailAddress: emailAddress? emailAddress : foundPosition && foundPosition.emails[0]? foundPosition.emails[0].value : '',
    //     linkedinUrl: linkedinUrl? linkedinUrl : foundPerson.linkedin.linkedinUrl,
    //     firstName: firstName? firstName : foundPerson.name.first,
    //     lastName: lastName? lastName : foundPerson.name.last,
    //     organization: organization? organization : foundPosition? foundPosition.companyInfo.name : '',
    //     title: foundPosition? foundPosition.title : '',
    //     phone: foundPerson.personalPhones[0] && foundPerson.personalPhones[0].status !== 'Suppressed' ? foundPerson.personalPhones[0].value : '',
    //   };
    // }//ﬁ





    // // If no person was found, then try and look up the organization by itself.
    // let employer;
    // if (foundPosition) {
    //   employer = {
    //     organization: organization? organization : foundPosition.companyInfo.name || '',
    //     numberOfEmployees: foundPosition.companyInfo.numberOfEmployees || 0,
    //     emailDomain:( foundPosition.companyInfo.domain? foundPosition.companyInfo.domain : emailDomain )|| '',
    //     linkedinCompanyPageUrl: foundPosition.companyInfo.linkedinUrl || '',
    //     technologies: foundPosition.companyInfo.technologies? foundPosition.companyInfo.technologies
    //       .filter((tech) => tech.category.match(RX_TECHNOLOGY_CATEGORIES))
    //       .map((tech) => ({ name: tech.name, category: tech.category })) : []
    //   };
    // } else {
    //   let report = await sails.helpers.http.get('https://api.leadiq.com/graphql', {
    //     query: `{ searchCompany(input: {
    //       ${organization? 'name: '+ JSON.stringify(organization) : ''}
    //       ${emailDomain? 'domain: '+ JSON.stringify(emailDomain) : ''}
    //     }) {
    //         totalResults
    //         results {
    //           name
    //           domain
    //           country
    //           address
    //           numberOfEmployees
    //           linkedinUrl
    //           technologies { name category parentCategory attributes categories }
    //         }
    //       }
    //     }`
    //   }, {
    //     Authorization: `Basic ${sails.config.custom.iqSecret}`,
    //     'content-type': 'application/json'
    //   });
    //   // sails.log('company search report:',report);

    //   if (report.errors) {
    //     sails.log.warn('Errors returned from IQ API when attempting to search directly for a matching organization:',report.errors);
    //   }
    //   let foundEmployer = report.data.searchCompany.results[0]; //sails.log(foundEmployer);
    //   if (foundEmployer) {
    //     employer = {
    //       organization: organization? organization : foundEmployer.name || '',
    //       numberOfEmployees: foundEmployer.numberOfEmployees || 0,
    //       emailDomain: emailDomain? emailDomain : foundEmployer.domain || '',
    //       linkedinCompanyPageUrl: foundEmployer.linkedinUrl || '',
    //       technologies: foundEmployer.technologies? foundEmployer.technologies
    //         .filter((tech) => tech.category.match(RX_TECHNOLOGY_CATEGORIES))
    //         .map((tech) => ({ name: tech.name, category: tech.category })) : []
    //     };// process.stdout.write(JSON.stringify(employer.technologies,0,2));
    //   }
    // }//ﬁ

    // return {
    //   person: undefined,
    //   employer: undefined
    // };

  }


};

