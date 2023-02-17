module.exports = {


  friendlyName: 'Send data to vanta',


  description: 'An hourly script that gathers and formats host and user data from Fleet instances sends it to Vanta on behalf of a Vanta user.',


  fn: async function () {

    // Find all VantaConnection records with isConnectedToVanta: true
    let allActiveVantaConnections = await VantaConnection.find({isConnectedToVanta: true});

    sails.log('Syncing Fleet instance data with Vanta for '+allActiveVantaConnections.length+(allActiveVantaConnections.length > 1 ? ' connections.' : ' connection.'));

    // Create an empty object to store caught errors. We don't want this script to stop running if there is an error with a single Vanta integration, so instead, we'll store any errors that occur and bail early for that connection if any occur, and we'll log them individually before the script is done.
    let errorReportById = {};

    // use sails.helpers.flow.simutaniouslyForEach to send data for each connection record.
    await sails.helpers.flow.simultaneouslyForEach(allActiveVantaConnections, async (vantaConnection)=>{
      let connectionIdAsString = String(vantaConnection.id);

      //  ┬─┐┌─┐┌─┐┬─┐┌─┐┌─┐┬ ┬  ╦  ╦┌─┐┌┐┌┌┬┐┌─┐  ┌┬┐┌─┐┬┌─┌─┐┌┐┌
      //  ├┬┘├┤ ├┤ ├┬┘├┤ └─┐├─┤  ╚╗╔╝├─┤│││ │ ├─┤   │ │ │├┴┐├┤ │││
      //  ┴└─└─┘└  ┴└─└─┘└─┘┴ ┴   ╚╝ ┴ ┴┘└┘ ┴ ┴ ┴   ┴ └─┘┴ ┴└─┘┘└┘
      let authorizationTokenRefreshResponse = await sails.helpers.http.post.with({
        url:'https://api.vanta.com/oauth/token',
        data: {
          client_id: sails.config.custom.vantaAuthorizationClientId, // eslint-disable-line camelcase
          client_secret: sails.config.custom.vantaAuthorizationClientSecret,// eslint-disable-line camelcase
          refresh_token: vantaConnection.vantaRefreshToken,// eslint-disable-line camelcase
          grant_type: 'refresh_token',// eslint-disable-line camelcase
        },
        headers: { accept: 'application/json' }
      }).tolerate((err)=>{
        // If an error occurs while sending a request to Vanta, we'll add the error to the errorReportById object, with this connections ID set as the key.
        errorReportById[connectionIdAsString] = new Error(`Could not refresh the token for Vanta connection (id: ${connectionIdAsString}). Full error: ${err}`);
      });

      if(errorReportById[connectionIdAsString]){// If there was an error with the previous request, bail early for this Vanta connection.
        return;
      }

      // Save the new authorization and refresh tokens in the database.
      let updatedRecord = await VantaConnection.updateOne({ id: vantaConnection.id }).set({
        vantaAuthToken: authorizationTokenRefreshResponse.access_token,
        vantaAuthTokenExpiresAt: Date.now() + (authorizationTokenRefreshResponse.expires_in * 1000),
        vantaRefreshToken: authorizationTokenRefreshResponse.refresh_token,
      });

      //  ╔═╗┬  ┌─┐┌─┐┌┬┐  ┬ ┬┌─┐┌─┐┬─┐┌─┐
      //  ╠╣ │  ├┤ ├┤  │   │ │└─┐├┤ ├┬┘└─┐
      //  ╚  ┴─┘└─┘└─┘ ┴   └─┘└─┘└─┘┴└─└─┘
      // Request user data from the Fleet instance to send to Vanta.
      let responseFromUserEndpoint = await sails.helpers.http.get(
        updatedRecord.fleetInstanceUrl + '/api/v1/fleet/users',
        {},
        {'Authorization': 'Bearer '+updatedRecord.fleetApiKey }
      )
      .tolerate((err)=>{// If an error occurs while sending a request to the Fleet instance, we'll add the error to the errorReportById object, with this connections ID set as the key.
        errorReportById[connectionIdAsString] = new Error(`When sending a request to the /users endpoint of a Fleet instance for a VantaConnection (id: ${connectionIdAsString}), the Fleet instance returned an Error: ${err}`);
      });

      if(errorReportById[connectionIdAsString]){// If there was an error with the previous request, bail early for this Vanta connection.
        return;
      }

      let usersToSyncWithVanta = [];
      // Iterate through the users list, creating user objects to send to Vanta.
      for(let user of responseFromUserEndpoint.users) {

        let authMethod;
        let mfaEnabled = false;
        let mfaMethods = [];
        if(user.sso_enabled && !user.api_only) { // If the user has sso_enabled: true, set the authMethod to SSO.
          authMethod = 'SSO';
        } else if(user.api_only) { // If the user is an api-only user, set the authMethod to 'TOKEN'
          authMethod = 'TOKEN';
          mfaMethods = ['UNSUPPORTED'];
        } else {// Otherwise, set the authMethod to 'PASSWORD'
          authMethod = 'PASSWORD';
          mfaMethods = ['DISABLED'];
        }

        // Set the permissionLevel using the user's global_role value.
        let permissionLevel = 'BASE';
        if(user.global_role === 'admin'){
          permissionLevel = 'ADMIN';
        } else if(user.global_role === 'maintainer'){
          permissionLevel = 'EDITOR';
        }

        // Create a user object for this Fleet user.
        let userToSyncWithVanta = {
          displayName: user.name,
          uniqueId: String(user.id),// Vanta requires this value to be a string.
          fullName: user.name,
          accountName: user.name,
          email: user.email,
          createdTimestamp: user.created_at,
          status: 'ACTIVE',// Always set to 'ACTIVE', if a user is removed from the Fleet instance, it will not be included in the response from the Fleet instance's /users endpoint.
          mfaEnabled,
          mfaMethods,
          externalUrl: vantaConnection.fleetInstanceUrl,// Setting externalUrl (Required by Vanta) for all users to be the Fleet instance url.
          authMethod,
          permissionLevel,
        };

        // Add the user object to the array of users to sync with Vanta.
        usersToSyncWithVanta.push(userToSyncWithVanta);
      }


      //  ┌─┐┌─┐┌┬┐  ┬ ┬┌─┐┌─┐┌┬┐┌─┐
      //  │ ┬├┤  │   ├─┤│ │└─┐ │ └─┐
      //  └─┘└─┘ ┴   ┴ ┴└─┘└─┘ ┴ └─┘
      // Get all hosts on the Fleet instance.
      let pageNumberForPossiblePaginatedResults = 0;
      let numberOfHostsPerRequest = 100;
      let allHostsOnThisFleetInstance = [];

      // Start sending requests to the Fleet instance's /hosts endpoint until we have all hosts.
      await sails.helpers.flow.until(async ()=>{
        let getHostsResponse = await sails.helpers.http.get(
          `${updatedRecord.fleetInstanceUrl}/api/v1/fleet/hosts?per_page=${numberOfHostsPerRequest}&page=${pageNumberForPossiblePaginatedResults}`,
          {},
          {'Authorization': 'bearer '+updatedRecord.fleetApiKey},
        );
        // Add the results to the allHostsOnThisFleetInstance array.
        allHostsOnThisFleetInstance = allHostsOnThisFleetInstance.concat(getHostsResponse.hosts);
        // Increment the page of results we're requesting.
        pageNumberForPossiblePaginatedResults++;
        // If we recieved less results than we requested, we've reached the last page of the results.
        return getHostsResponse.hosts.length !== numberOfHostsPerRequest;
      }, 10000)
      .tolerate(()=>{// If an error occurs while sending a request to the Fleet instance, we'll add the error to the errorReportById object, with this connections ID set as the key.
        errorReportById[connectionIdAsString] = new Error(`When requesting all hosts from a Fleet instance for a VantaConnection (id: ${connectionIdAsString}), the Fleet instance did not respond with all of it's hosts in the set amount of time.`);
      });

      if(errorReportById[connectionIdAsString]){// If an error occured in the previous request, we'll bail early for this connection.
        return;
      }

      let macOsHosts = allHostsOnThisFleetInstance.filter((host)=>{
        return host.platform === 'darwin';
      });

      let macHostsToSyncWithVanta = [];


      await sails.helpers.flow.simultaneouslyForEach(macOsHosts, async (host) => {
        let hostIdAsString = String(host.id);
        // Start building the host resource to send to Vanta, using information we get from the Fleet instance's get Hosts endpoint
        let macOsHostToSyncWithVanta = {
          displayName: host.display_name,
          uniqueId: hostIdAsString,
          externalUrl: updatedRecord.fleetInstanceUrl + encodeURIComponent('/hosts/'+hostIdAsString),
          collectedTimestamp: host.updated_at,
          osName: 'macOS', // Setting the osName for all macOS hosts to 'macOS'. Different versions of macOS have different prefixes, (e.g., a macOS host running 12.6 would be returned as "macOS 12.6.1", while a mac running version 10.15.7 would be displayed as "Mac OS X 10.15.7")
          osVersion: host.os_version.replace(/^([\D]+)\s(.+)/g, '$2'), // removing everything but the version number (XX.XX.XX) from the host's os_version value.
          hardwareUuid: host.uuid,
          serialNumber: host.hardware_serial,
          applications: [],
          browserExtensions: [],
          drives: [],
          users: [],// Sending an empty array of users.
          systemScreenlockPolicies: [],// Sending an empty array of screenlock policies.
          isManaged: false, // Defaulting to false
          autoUpdatesEnabled: false, // Always sending this value as false
        };

        // Skip further details for pending enrollment MDM hosts as we don't yet have much
        // information about them and Vanta requires disk encryption and other information we can't
        // provide.
        if (host.mdm && host.mdm.enrollment_status === 'Pending') {
          return;
        }

        // Send a request to this host's API endpoint to get the required information about this host.
        let detailedInformationAboutThisHost = await sails.helpers.http.get(
          updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts/'+host.id,
          {},
          {'Authorization': 'bearer '+updatedRecord.fleetApiKey}
        )
        .retry()
        .intercept((err)=>{// If an error occurs while sending a request to the Fleet instance, we'll throw an error.
          return new Error(`When sending a request to the Fleet instance's /hosts/${host.id} endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${err}`);
        });

        // Build a drive object for this host, using the host's disk_encryption_enabled value to set the boolean values for `encrytped` and `filevaultEnabled`
        let driveInformationForThisHost = {
          name: 'Hard drive',
          encrypted: detailedInformationAboutThisHost.host.disk_encryption_enabled,
          filevaultEnabled: detailedInformationAboutThisHost.host.disk_encryption_enabled,
        };
        macOsHostToSyncWithVanta.drives.push(driveInformationForThisHost);

        // Iterate through the array of software on a host to populate this hosts applications and
        // browserExtensions arrays.
        const softwareList = detailedInformationAboutThisHost.host.software;
        if (softwareList) {
          for (let software of softwareList) {
            let softwareToAdd = {};
            if (software.source === 'firefox_addons') {
              softwareToAdd.name = software.name;
              softwareToAdd.browser = 'FIREFOX';
              softwareToAdd.extensionId = software.name + ' ' + software.version;// Set the extensionId to be the software's name and the software version.
              macOsHostToSyncWithVanta.browserExtensions.push(softwareToAdd);
            } else if (software.source === 'chrome_extensions') {
              softwareToAdd.name = software.name;
              softwareToAdd.extensionId = software.name + ' ' + software.version;
              softwareToAdd.browser = 'CHROME';
              macOsHostToSyncWithVanta.browserExtensions.push(softwareToAdd);
            } else if (software.source === 'apps') {
              softwareToAdd.name = software.name + ' ' + software.version;
              softwareToAdd.bundleId = software.bundle_identifier ? software.bundle_identifier : ' '; // If the software is missing a bundle identifier, we'll set it to a blank string.
              macOsHostToSyncWithVanta.applications.push(softwareToAdd);
            }
          }
        }

        // Add the host to the array of macOS hosts to sync with Vanta
        macHostsToSyncWithVanta.push(macOsHostToSyncWithVanta);
      }).tolerate((err)=>{// If an error occurs while sending requests for each host, add the error to the errorReportById object.
        errorReportById[connectionIdAsString] = new Error(`When building an array of macOS hosts for a Vanta connection (id: ${connectionIdAsString}), an error occured: ${err}`);
      });// After every macOS host

      if(errorReportById[connectionIdAsString]){// If an error occured while gathering detailed host information, we'll bail early for this connection.
        return;
      }


      //  ┌─┐┬ ┬┌┐┌┌─┐  ┬ ┬┌─┐┌─┐┬─┐┌─┐  ┬ ┬┬┌┬┐┬ ┬  ╦  ╦┌─┐┌┐┌┌┬┐┌─┐
      //  └─┐└┬┘││││    │ │└─┐├┤ ├┬┘└─┐  ││││ │ ├─┤  ╚╗╔╝├─┤│││ │ ├─┤
      //  └─┘ ┴ ┘└┘└─┘  └─┘└─┘└─┘┴└─└─┘  └┴┘┴ ┴ ┴ ┴   ╚╝ ┴ ┴┘└┘ ┴ ┴ ┴
      await sails.helpers.http.sendHttpRequest.with({
        method: 'PUT',
        url: 'https://api.vanta.com/v1/resources/user_account/sync_all',
        body: {
          sourceId: vantaConnection.sourceId,
          resourceId: '63868a88d436911435a94035',
          resources: usersToSyncWithVanta,
        },
        headers: {
          'accept': 'application/json',
          'authorization': 'Bearer '+updatedRecord.vantaAuthToken,
          'content-type': 'application/json',
        },
      }).tolerate((err)=>{// If an error occurs while sending a request to Vanta, we'll add the error to the errorReportById object, with this connections ID set as the key.
        errorReportById[connectionIdAsString] = new Error(`vantaError: When sending a PUT request to the Vanta's '/user_account/sync_all' endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${err}`);
      });

      if(errorReportById[connectionIdAsString]){// If an error occured in the previous request, we'll bail early for this connection.
        return;
      }

      //  ┌─┐┬ ┬┌┐┌┌─┐  ┬ ┬┌─┐┌─┐┌┬┐┌─┐  ┬ ┬┬┌┬┐┬ ┬  ╦  ╦┌─┐┌┐┌┌┬┐┌─┐
      //  └─┐└┬┘││││    ├─┤│ │└─┐ │ └─┐  ││││ │ ├─┤  ╚╗╔╝├─┤│││ │ ├─┤
      //  └─┘ ┴ ┘└┘└─┘  ┴ ┴└─┘└─┘ ┴ └─┘  └┴┘┴ ┴ ┴ ┴   ╚╝ ┴ ┴┘└┘ ┴ ┴ ┴
      await sails.helpers.http.sendHttpRequest.with({
        method: 'PUT',
        url: 'https://api.vanta.com/v1/resources/macos_user_computer/sync_all',
        body: {
          sourceId: vantaConnection.vantaSourceId,
          resourceId: '63868a569c18bd7adc6b7907',
          resources: macHostsToSyncWithVanta,
        },
        headers: {
          'accept': 'application/json',
          'authorization': 'Bearer '+updatedRecord.vantaAuthToken,
          'content-type': 'application/json',
        },
      }).tolerate((err)=>{// If an error occurs while sending a request to Vanta, we'll add the error to the errorReportById object, with this connections ID set as the key.
        errorReportById[connectionIdAsString] = new Error(`vantaError: When sending a PUT request to the Vanta's '/macos_user_computer/sync_all' endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${err}`);
      });

      if(errorReportById[connectionIdAsString]){// If an error occured in the previous request, we'll bail early for this connection.
        return;
      }

      errorReportById[connectionIdAsString] = false;
    });//∞ After every active VantaConnection

    let numberOfLoggedErrors = 0;

    // After we've sent requests for all active Vanta connections, log any errors that occured.
    for (let connectionIdAsString of Object.keys(errorReportById)) {
      if (false === errorReportById[connectionIdAsString]) {
        // If data was sent to Vanta sucessfully, do nothing.
      } else {
        // If an error was logged for a VantaConnection, log the error, and increment the numberOfLoggedErrors
        numberOfLoggedErrors++;
        sails.log.warn('An error occurred while syncing the vanta connection for VantaCustomer with id '+connectionIdAsString+'. Logged error:\n'+errorReportById[connectionIdAsString]);
      }
    }//∞

    sails.log('Information has been sent to Vanta for '+(allActiveVantaConnections.length - numberOfLoggedErrors)+(allActiveVantaConnections.length - numberOfLoggedErrors > 1 || numberOfLoggedErrors === allActiveVantaConnections.length ? ' connections.' : ' connection.'));

  }


};

