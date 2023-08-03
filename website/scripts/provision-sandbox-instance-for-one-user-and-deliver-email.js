module.exports = {


  friendlyName: 'Provision Sandbox instance for one user and deliver email.',


  description: 'Provisions a new Fleet Sandbox instance for a user on the Fleet Sandbox waitlist, and sends an email to the user.',

  extendedDescription: 'This script will provision a Sandbox instance for the user who has been on the waitlist the longest.',


  fn: async function () {


    let earliestCreatedUserCurrentlyOnWaitlist = await User.find({inSandboxWaitlist: true})
    .limit(1)
    .sort('createdAt ASC');

    // If there are no users on the Fleet sandbox waitlist, end the script.
    if(earliestCreatedUserCurrentlyOnWaitlist.length === 0){
      sails.log('There are no users currently waiting on the Fleet Sandbox Waitlist.');
      return;
    }

    let userToRemoveFromSandboxWaitlist = earliestCreatedUserCurrentlyOnWaitlist[0];

    const FIVE_DAYS_IN_MS = (5*24*60*60*1000);
    // Creating an expiration JS timestamp for the Fleet sandbox instance. NOTE: We send this value to the cloud provisioner API as an ISO 8601 string.
    let fleetSandboxExpiresAt = Date.now() + FIVE_DAYS_IN_MS;

    // Creating a fleetSandboxDemoKey, this will be used for the user's password when we log them into their Sandbox instance.
    let fleetSandboxDemoKey = await sails.helpers.strings.uuid();

    // Send a POST request to the cloud provisioner API
    let cloudProvisionerResponseData = await sails.helpers.http.post(
      'https://sandbox.fleetdm.com/new',
      { // Request body
        'name': userToRemoveFromSandboxWaitlist.firstName + ' ' + userToRemoveFromSandboxWaitlist.lastName,
        'email': userToRemoveFromSandboxWaitlist.emailAddress,
        'password': fleetSandboxDemoKey, //« this provisioner API was originally designed to accept passwords, but rather than specifying the real plaintext password, since users always access Fleet Sandbox from their fleetdm.com account anyway, this generated demo key is used instead to avoid any confusion
        'sandbox_expiration': new Date(fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
      },
      { // Request headers
        'Authorization':sails.config.custom.cloudProvisionerSecret
      }
    )
    .timeout(10000)
    .intercept(['requestFailed', 'non200Response'], (err)=>{
      // If we received a non-200 response from the cloud provisioner API, we'll throw a 500 error.
      return new Error('When attempting to provision a Sandbox instance for a user on the Fleet Sandbox waitlist ('+userToRemoveFromSandboxWaitlist.emailAddress+'), the cloud provisioner gave a non 200 response. Raw response received from provisioner: '+err.stack);
    })
    .intercept({name: 'TimeoutError'},(err)=>{
      // If the request timed out, log a warning and return a 'requestToSandboxTimedOut' response.
      sails.log.warn('When attempting to provision a Sandbox instance for a user on the Fleet Sandbox waitlist ('+userToRemoveFromSandboxWaitlist.emailAddress+'), the request to the cloud provisioner took over timed out. Raw error: '+err.stack);
      return 'requestToSandboxTimedOut';
    });

    if(!cloudProvisionerResponseData.URL) {
      // If we didn't receive a URL in the response from the cloud provisioner API, we'll throwing an error before we save the new user record and the user will need to try to sign up again.
      throw new Error(
        `When provisioning a Fleet Sandbox instance for a user on the Fleet Sandbox waitlist (${userToRemoveFromSandboxWaitlist.emailAddress}), the response data from the cloud provisioner API was malformed. It did not contain a valid Fleet Sandbox instance URL in its expected "URL" property.
        Here is the malformed response data (parsed response body) from the cloud provisioner API: ${cloudProvisionerResponseData}`
      );
    }

    // Start polling the /healthz endpoint of the created Fleet Sandbox instance, once it returns a 200 response, we'll continue.
    await sails.helpers.flow.until( async()=>{
      let healthCheckResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponseData.URL+'/healthz')
      .timeout(5000)
      .tolerate('non200Response')
      .tolerate('requestFailed')
      .tolerate({name: 'TimeoutError'});
      if(healthCheckResponse) {
        return true;
      }
    }, 10000).intercept('tookTooLong', ()=>{
      return new Error('This newly provisioned Fleet Sandbox instance (for '+userToRemoveFromSandboxWaitlist.emailAddress+') is taking too long to respond with a 2xx status code, even after repeatedly polling the health check endpoint.  Note that failed requests and non-2xx responses from the health check endpoint were ignored during polling.  Search for a bit of non-dynamic text from this error message in the fleetdm.com source code for more info on exactly how this polling works.');
    });


    await User.updateOne({id: userToRemoveFromSandboxWaitlist.id})
    .set({
      fleetSandboxURL: cloudProvisionerResponseData.URL,
      fleetSandboxExpiresAt,
      fleetSandboxDemoKey,
      inSandboxWaitlist: false,
    });


    // Send the user an email to let them know that their Fleet sandbox instance is ready.
    await sails.helpers.sendTemplateEmail.with({
      to: userToRemoveFromSandboxWaitlist.emailAddress,
      from: sails.config.custom.fromEmailAddress,
      fromName: sails.config.custom.fromName,
      subject: 'Your Fleet Sandbox instance is ready!',
      template: 'email-sandbox-ready-approved',
      templateData: {},
    });


  }


};

