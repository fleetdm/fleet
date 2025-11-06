module.exports = {


  friendlyName: 'Send entra heartbeat requests',


  description: 'Sends heartbeat requests to Microsoft compliance tenants to keep the integration active.',


  fn: async function () {

    // Find all MicrosoftComplianceTenant records with setupComplete: true
    let allActiveEntraTenants = await MicrosoftComplianceTenant.find({setupCompleted: true});

    sails.log('Syncing hearbeat requests for '+allActiveEntraTenants.length+(allActiveEntraTenants.length > 1 ? ' tenants.' : ' tenant.'));

    // Create an empty object to store caught errors. We don't want this script to stop running if there is an error with a single entra tenant, so instead, we'll store any errors that occur and bail early for that tenant if any occur, and we'll log them individually before the script is done.
    let errorReportById = {};

    await sails.helpers.flow.simultaneouslyForEach(allActiveEntraTenants, async (entraTenant)=>{
      let connectionIdAsString = String(entraTenant.id);

      let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
        complianceTenantRecordId: entraTenant.id
      }).tolerate((err)=>{
        errorReportById[connectionIdAsString] = new Error(`Could not get an access token and API urls for a MicrosoftComplianceTenant (id: ${connectionIdAsString}). Full error: ${err}`);
      });

      if(errorReportById[connectionIdAsString]){// If there was an error with the previous request, bail early for this Entra tenant.
        return;
      }

      let accessToken = tokenAndApiUrls.manageApiAccessToken;
      let tenantDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;

      let isoTimestampForThisRequest = new Date().toISOString();
      // Send a heartbeat request.
      let tenantHeartbeatResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'PUT',
        url: `${tenantDataSyncUrl}/PartnerTenantHeartbeat(guid'${encodeURIComponent(entraTenant.entraTenantId)}')?api-version=1.6`,
        headers: {
          'Authorization': `Bearer ${accessToken}`,
          'Content-Type': 'application/json',
          'Prefer': 'return-content'
        },
        body: {
          Timestamp: isoTimestampForThisRequest,
        }
      }).tolerate((err)=>{
        errorReportById[connectionIdAsString] = new Error(`Could not send a heartbeat request for a MicrosoftComplianceTenant (id: ${connectionIdAsString}). Full error: ${require('util').inspect(err, {depth: null})}`);
      });
      if(errorReportById[connectionIdAsString]){// If there was an error with the previous request, bail early for this Entra tenant.
        return;
      }
      let parsedtenantHeartbeatResponse;
      try {
        parsedtenantHeartbeatResponse = JSON.parse(tenantHeartbeatResponse.body);
      } catch(err){
        errorReportById[connectionIdAsString] = new Error(`Could not parse the JSON response of a heartbeat request for a Microsoft compliance tenant (id: ${connectionIdAsString}). Full error: ${require('util').inspect(err, {depth: null})}`);
      }

      if(errorReportById[connectionIdAsString]){// If there was an error with the previous request, bail early for this Entra tenant.
        return;
      }

      if(parsedtenantHeartbeatResponse.ResyncTimestamp){
        // TODO: do we want to do anything about the resync timestamp if it is set?
      }

      await MicrosoftComplianceTenant.updateOne({id: entraTenant.id}).set({
        lastHeartbeatAt: Date.now(),
      });
    });

    let numberOfLoggedErrors = 0;

    // After we've sent requests for all active Entra tenants, log any errors that occured.
    for (let connectionIdAsString of Object.keys(errorReportById)) {
      if (false === errorReportById[connectionIdAsString]) {
        // If a heartbeat was sent successfully, do nothing.
      } else {
        // If an error was logged for a entra tenant, log the error, and increment the numberOfLoggedErrors
        numberOfLoggedErrors++;
        sails.log.warn('p1: An error occurred while sending a heartbeat request for a Microsfot entra compliance tenant with the id'+connectionIdAsString+'. Logged error:\n'+errorReportById[connectionIdAsString]);
      }
    }//∞

    sails.log('Heartbeat requests have been sent for '+(allActiveEntraTenants.length - numberOfLoggedErrors)+(allActiveEntraTenants.length - numberOfLoggedErrors > 1 || numberOfLoggedErrors === allActiveEntraTenants.length ? ' tenants.' : ' tenant.'));
  }


};

