module.exports = {


  friendlyName: 'Deliver gitops request submission',


  description: '',


  inputs: {
    firstName: { type: 'string', required: true },
    lastName: { type: 'string', required: true },
    emailAddress: { type: 'string', isEmail: true, required: true },
    location: { type: 'string', required: true },
    numberOfHosts: { type: 'string', required: true },
    managedPlatforms: { type: {}, required: true },
    willingToHost: { type: 'string'},
  },


  exits: {
    success: {
      description: 'A gitops workshop request was submitted.'
    },
    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },
  },


  fn: async function ({firstName, lastName, location, emailAddress, numberOfHosts, managedPlatforms, willingToHost}) {


    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForContactFormSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }


    // Convert the managedPlatforms object into a string.
    let platformFriendlyNamesByManagedPlatformValues = {
      macos: 'macOS',
      windows: 'Windows',
      linux: 'Linux',
      android: 'Android',
      iosOrIpados: 'iOS/iPadOS',
      chromeos: 'ChromeOS',
    };
    let managedPlatformsString = 'Selected platforms: ';
    for(let selectedPlatform of _.keysIn(managedPlatforms)){
      if(managedPlatforms[selectedPlatform] === true){
        managedPlatformsString += `\n\t- ${platformFriendlyNamesByManagedPlatformValues[selectedPlatform]}`;
      }
    }


    // Build a description with information from the form submission to add to the created/found contact record.
    let descriptionForCrmUpdate =
    `
    Submitted the gitops workshop request form.
    Submission information:
    They are ${willingToHost ? '' : 'not '}interested in hosting a workshop at their company's office.
    Location: ${location}
    Email: ${emailAddress}
    Number of hosts: ${numberOfHosts}
    ${managedPlatformsString}
    `;

    let attributionCookieOrUndefined = this.req.cookies.marketingAttribution;

    await sails.helpers.flow.build(async ()=>{
      let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: emailAddress,
        firstName: firstName,
        lastName: lastName,
        contactSource: 'Website - Contact forms',
        description: descriptionForCrmUpdate,
        marketingAttributionCookie: attributionCookieOrUndefined
      }).intercept((err)=>{
        return new Error(`Could not create/update a contact or account. Full error: ${require('util').inspect(err)}`);
      });

      // Add contact to campaign.
      await sails.helpers.salesforce.createCampaignMember.with({
        salesforceContactId: recordIds.salesforceContactId,
        salesforceCampaignId: '701UG00000bLCLpYAO',// 2026_01-FE-GitOps_Workshop_Interest Campaign
      });

    }).tolerate((err)=>{
      sails.log.warn(`When a user (${firstName} ${lastName}, email: ${emailAddress}) submitted the gitops workshop request form, an error occured when updating CRM records for this user.\n Submission information: ${descriptionForCrmUpdate.split('Submission information:')[1]}\n Full error: ${require('util').inspect(err)}`);
    });




    // All done.
    return;

  }


};
