const userAgent = 'Fleet/Vanta Updater';

module.exports = {


  friendlyName: 'Send data to vanta',


  description: 'An hourly script that gathers and formats host and user data from Fleet instances sends it to Vanta on behalf of a Vanta user.',


  fn: async function () {
    let util = require('util');

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
      })
      .retry({raw:{statusCode: 503}})// Retry requests that respond with "503: Service temporarily unavailable"
      .retry({raw:{statusCode: 504}})// Retry requests that respond with "504: Endpoint request timed out"
      .tolerate((err)=>{
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
      // Note: this request is in a try-catch block so we can handle errors sent from the retry() method
      let responseFromUserEndpoint;
      try {
        responseFromUserEndpoint = await sails.helpers.http.get(
          updatedRecord.fleetInstanceUrl + '/api/v1/fleet/users',
          {},
          {'Authorization': 'Bearer '+updatedRecord.fleetApiKey, 'User-Agent': userAgent }
        )
        .retry()
        .tolerate((err)=>{// If an error occurs while sending a request to the Fleet instance, we'll add the error to the errorReportById object, with this connections ID set as the key.
          errorReportById[connectionIdAsString] = new Error(`When sending a request to the /users endpoint of a Fleet instance for a VantaConnection (id: ${connectionIdAsString}), the Fleet instance returned an Error: ${util.inspect(err.raw)}`);
        });
      } catch(error) {
        errorReportById[connectionIdAsString] = new Error(`When sending a request to the /users endpoint of a Fleet instance for a VantaConnection (id: ${connectionIdAsString}), the Fleet instance returned an Error: ${util.inspect(error.raw)}`);
      }

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
          {'Authorization': 'bearer '+updatedRecord.fleetApiKey, 'User-Agent': userAgent},
        )
        .retry();
        // Add the results to the allHostsOnThisFleetInstance array.
        allHostsOnThisFleetInstance = allHostsOnThisFleetInstance.concat(getHostsResponse.hosts);
        // Increment the page of results we're requesting.
        pageNumberForPossiblePaginatedResults++;
        // If we recieved less results than we requested, we've reached the last page of the results.
        return getHostsResponse.hosts.length !== numberOfHostsPerRequest;
      }, 30000)
      .tolerate(()=>{// If an error occurs while sending a request to the Fleet instance, we'll add the error to the errorReportById object, with this connections ID set as the key.
        errorReportById[connectionIdAsString] = new Error(`When requesting all hosts from a Fleet instance for a VantaConnection (id: ${connectionIdAsString}), the Fleet instance did not respond with all of it's hosts in the set amount of time.`);
      });

      if(errorReportById[connectionIdAsString]){// If an error occured in the previous request, we'll bail early for this connection.
        return;
      }

      // If this is Fleet's Vanta connection, exclude hosts on the "Compliance exclusions" team.
      // See https://github.com/fleetdm/fleet/issues/19312 for more information.
      if(vantaConnection.id === 3){
        allHostsOnThisFleetInstance = allHostsOnThisFleetInstance.filter((host)=>{
          return host.team_id !== 178;// Compliance exclusions team
        });
      }

      let macOsHosts = allHostsOnThisFleetInstance.filter((host)=>{
        return host.platform === 'darwin';
      });

      let windowsHosts = allHostsOnThisFleetInstance.filter((host)=>{
        return host.platform === 'windows';
      });

      let macHostsToSyncWithVanta = [];
      let windowsHostsToSyncWithVanta = [];


      await sails.helpers.flow.simultaneouslyForEach(macOsHosts, async (host) => {
        let hostIdAsString = String(host.id);
        // Start building the host resource to send to Vanta, using information we get from the Fleet instance's get Hosts endpoint
        let macOsHostToSyncWithVanta = {
          displayName: host.display_name,
          uniqueId: hostIdAsString,
          externalUrl: updatedRecord.fleetInstanceUrl + '/hosts/'+encodeURIComponent(hostIdAsString),
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

        // If the host has an mdm property, set the `isManaged` parameter to true if the hosts's mdm enrollment_status is either "On (automatic)" or "On (manual)")
        if(host.mdm !== undefined && host.mdm.enrollment_status !== null) {
          if(host.mdm.enrollment_status === 'On (automatic)' || host.mdm.enrollment_status === 'On (manual)'){
            macOsHostToSyncWithVanta.isManaged = true;
          }
          // If this host's MDM status is pending (MDM is not yet fully turned on for this host), then it doesn't have comprehensive vitals nor a complete host details page thus we'll exclude it from the hosts we sync with Vanta.
          // More info about pending hosts: https://github.com/fleetdm/fleet/blob/3cc7c971c2c24e28d06323c329475ae32e9a8198/docs/Using-Fleet/MDM-setup.md#pending-hosts
          if(host.mdm.enrollment_status === 'Pending') {
            return;
          }
        }

        // Send a request to this host's API endpoint to get the required information about this host.
        let detailedInformationAboutThisHost = await sails.helpers.http.get(
          updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts/'+encodeURIComponent(hostIdAsString),
          {},
          {'Authorization': 'bearer '+updatedRecord.fleetApiKey, 'User-Agent': userAgent}
        )
        .retry()
        .intercept((err)=>{// If an error occurs while sending a request to the Fleet instance, we'll throw an error.
          return new Error(`When sending a request to the Fleet instance's /hosts/${host.id} endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${util.inspect(err.raw)}`);
        });

        if(!detailedInformationAboutThisHost.host) {
          throw new Error(`When sending a request to the Fleet instance's /hosts/${host.id} endpoint for a Vanta connection (id: ${connectionIdAsString}), the response from the Fleet API did not include a host. Response from the Fleet API: ${util.inspect(detailedInformationAboutThisHost)}`);
        }

        if (detailedInformationAboutThisHost.host.disk_encryption_enabled !== undefined && detailedInformationAboutThisHost.host.disk_encryption_enabled !== null) {
          // Build a drive object for this host, using the host's disk_encryption_enabled value to set the boolean values for `encrytped` and `filevaultEnabled`
          let driveInformationForThisHost = {
            name: 'Hard drive',
            encrypted: detailedInformationAboutThisHost.host.disk_encryption_enabled,
            filevaultEnabled: detailedInformationAboutThisHost.host.disk_encryption_enabled,
          };
          macOsHostToSyncWithVanta.drives.push(driveInformationForThisHost);
        }

        // Iterate through the array of software on a host to populate this hosts applications and
        // browserExtensions arrays.
        const softwareList = detailedInformationAboutThisHost.host.software;
        if (softwareList) {
          for (let software of softwareList) {
            let softwareToAdd = {
              name: software.name,
            };
            if(software.source === 'firefox_addons' || software.source === 'chrome_extensions') {
              softwareToAdd.browser = software.source.toUpperCase().split('_')[0];// Get the uppercased first word of the software source, this will either be CHROME or FIREFOX.
              softwareToAdd.extensionId = software.name + ' ' + software.version;// Set the extensionId to be the software's name and the software version.
              if(software.extension_id !== undefined && software.extension_id !== null) {// If the Fleet instance reported an extension_id for the extension, we'll use that value.
                softwareToAdd.extensionId = software.extension_id;
              }
              macOsHostToSyncWithVanta.browserExtensions.push(softwareToAdd);
            } else if(software.source === 'apps'){
              softwareToAdd.name += ' '+software.version;// Add the version to the software name
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


      await sails.helpers.flow.simultaneouslyForEach(windowsHosts, async (host) => {
        let hostIdAsString = String(host.id);
        // Start building the host resource to send to Vanta, using information we get from the Fleet instance's get Hosts endpoint
        let windowsHostToSyncWithVanta = {
          displayName: host.display_name,
          uniqueId: hostIdAsString,
          externalUrl: updatedRecord.fleetInstanceUrl + '/hosts/'+encodeURIComponent(hostIdAsString),
          collectedTimestamp: host.updated_at,
          osName: 'Windows',
          osVersion: host.code_name,
          hardwareUuid: host.uuid,
          serialNumber: host.hardware_serial,
          programs: [],
          browserExtensions: [],
          drives: [],
          users: [],
          systemScreenlockPolicies: [],
          isManaged: false, // Defaulting to false, if the host is enrolled in an MDM, we'll set this value to true.
          autoUpdatesEnabled: false, // Defaulting this value to false.
          lastEnrolledTimestamp: host.last_enrolled_at,
        };

        // If the host has an mdm property, set the `isManaged` parameter to true if the hosts's mdm enrollment_status is either "On (automatic)" or "On (manual)")
        if(host.mdm !== undefined && host.mdm.enrollment_status !== null){
          if(host.mdm.enrollment_status === 'On (automatic)' || host.mdm.enrollment_status === 'On (manual)'){
            windowsHostToSyncWithVanta.isManaged = true;
          }
        }

        // Send a request to this host's API endpoint to get the required information about this host.
        let detailedInformationAboutThisHost = await sails.helpers.http.get(
          updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts/'+encodeURIComponent(hostIdAsString),
          {},
          {'Authorization': 'bearer '+updatedRecord.fleetApiKey, 'User-Agent': userAgent}
        )
        .retry()
        .intercept((err)=>{// If an error occurs while sending a request to the Fleet instance, we'll throw an error.
          return new Error(`When sending a request to the Fleet instance's /hosts/${host.id} endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${util.inspect(err.raw)}`);
        });

        if(!detailedInformationAboutThisHost.host){
          throw new Error(`When sending a request to the Fleet instance's /hosts/${host.id} endpoint for a Vanta connection (id: ${connectionIdAsString}), the response from the Fleet API did not include a host. Response from the Fleet API: ${util.inspect(detailedInformationAboutThisHost)}`);
        }

        if (detailedInformationAboutThisHost.host.disk_encryption_enabled !== undefined && detailedInformationAboutThisHost.host.disk_encryption_enabled !== null) {
          // Build a drive object for this host, using the host's disk_encryption_enabled value to set the boolean values for `encrytped` and `filevaultEnabled`
          let driveInformationForThisHost = {
            name: 'Hard drive',
            encrypted: detailedInformationAboutThisHost.host.disk_encryption_enabled,
          };
          windowsHostToSyncWithVanta.drives.push(driveInformationForThisHost);
        }

        // Iterate through the array of software on a host to populate this hosts applications and
        // browserExtensions arrays.
        const softwareList = detailedInformationAboutThisHost.host.software;
        if (softwareList) {
          for (let software of softwareList) {
            let softwareToAdd = {
              name: software.name,
            };
            if (software.source === 'firefox_addons' || software.source === 'chrome_extensions') {
              softwareToAdd.browser = software.source.toUpperCase().split('_')[0];// Get the uppercased first word of the software source, this will either be CHROME or FIREFOX.
              softwareToAdd.extensionId = software.name + ' ' + software.version;// Set the extensionId to be the software's name and the software version.
              if(software.extension_id !== undefined && software.extension_id !== null) {// If the Fleet instance reported an extension_id for this extension, we'll use that value.
                softwareToAdd.extensionId = software.extension_id;
              }
              windowsHostToSyncWithVanta.browserExtensions.push(softwareToAdd);
            } else if (software.source === 'programs') {
              windowsHostToSyncWithVanta.programs.push(softwareToAdd);
            }
          }
        }
        // Add the host to the array of macOS hosts to sync with Vanta
        windowsHostsToSyncWithVanta.push(windowsHostToSyncWithVanta);
      }).tolerate((err)=>{// If an error occurs while sending requests for each host, add the error to the errorReportById object.
        errorReportById[connectionIdAsString] = new Error(`When building an array of Windows hosts for a Vanta connection (id: ${connectionIdAsString}), an error occured: ${err}`);
      });// After every Windows host

      if(errorReportById[connectionIdAsString]){// If an error occured while gathering detailed host information, we'll bail early for this connection.
        return;
      }
      //  ┌─┐┬ ┬┌┐┌┌─┐  ┬ ┬┌─┐┌─┐┬─┐┌─┐  ┬ ┬┬┌┬┐┬ ┬  ╦  ╦┌─┐┌┐┌┌┬┐┌─┐
      //  └─┐└┬┘││││    │ │└─┐├┤ ├┬┘└─┐  ││││ │ ├─┤  ╚╗╔╝├─┤│││ │ ├─┤
      //  └─┘ ┴ ┘└┘└─┘  └─┘└─┘└─┘┴└─└─┘  └┴┘┴ ┴ ┴ ┴   ╚╝ ┴ ┴┘└┘ ┴ ┴ ┴
      // Note: this request is in a try-catch block so we can handle errors sent from the retry() method
      try {
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
        })
        .retry();
      } catch(error) {
        errorReportById[connectionIdAsString] = new Error(`vantaError: When sending a PUT request to the Vanta's '/user_account/sync_all' endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${util.inspect(error.raw)}`);
      }

      if(errorReportById[connectionIdAsString]){// If an error occured in the previous request, we'll bail early for this connection.
        return;
      }

      //  ╔═╗┬ ┬┌┐┌┌─┐  ┌┬┐┌─┐┌─┐╔═╗╔═╗  ┬ ┬┌─┐┌─┐┌┬┐┌─┐  ┬ ┬┬┌┬┐┬ ┬  ╦  ╦┌─┐┌┐┌┌┬┐┌─┐
      //  ╚═╗└┬┘││││    │││├─┤│  ║ ║╚═╗  ├─┤│ │└─┐ │ └─┐  ││││ │ ├─┤  ╚╗╔╝├─┤│││ │ ├─┤
      //  ╚═╝ ┴ ┘└┘└─┘  ┴ ┴┴ ┴└─┘╚═╝╚═╝  ┴ ┴└─┘└─┘ ┴ └─┘  └┴┘┴ ┴ ┴ ┴   ╚╝ ┴ ┴┘└┘ ┴ ┴ ┴
      // Note: this request is in a try-catch block so we can handle errors sent from the retry() method
      try {
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
        })
        .retry();
      } catch (error) {
        errorReportById[connectionIdAsString] = new Error(`vantaError: When sending a PUT request to the Vanta's '/macos_user_computer/sync_all' endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${util.inspect(error.raw)}`);
      }

      if(errorReportById[connectionIdAsString]){// If an error occured in the previous request, we'll bail early for this connection.
        return;
      }

      //  ╔═╗┬ ┬┌┐┌┌─┐  ╦ ╦┬┌┐┌┌┬┐┌─┐┬ ┬┌─┐  ┬ ┬┌─┐┌─┐┌┬┐┌─┐  ┬ ┬┬┌┬┐┬ ┬  ╦  ╦┌─┐┌┐┌┌┬┐┌─┐
      //  ╚═╗└┬┘││││    ║║║││││ │││ ││││└─┐  ├─┤│ │└─┐ │ └─┐  ││││ │ ├─┤  ╚╗╔╝├─┤│││ │ ├─┤
      //  ╚═╝ ┴ ┘└┘└─┘  ╚╩╝┴┘└┘─┴┘└─┘└┴┘└─┘  ┴ ┴└─┘└─┘ ┴ └─┘  └┴┘┴ ┴ ┴ ┴   ╚╝ ┴ ┴┘└┘ ┴ ┴ ┴
      // Note: this request is in a try-catch block so we can handle errors sent from the retry() method
      try {
        await sails.helpers.http.sendHttpRequest.with({
          method: 'PUT',
          url: 'https://api.vanta.com/v1/resources/windows_user_computer/sync_all',
          body: {
            sourceId: vantaConnection.vantaSourceId,
            resourceId: '64012c3f4bd5adc73b133459',
            resources: windowsHostsToSyncWithVanta,
          },
          headers: {
            'accept': 'application/json',
            'authorization': 'Bearer '+updatedRecord.vantaAuthToken,
            'content-type': 'application/json',
          },
        })
        .retry();
      } catch (error) {
        errorReportById[connectionIdAsString] = new Error(`vantaError: When sending a PUT request to the Vanta's '/macos_user_computer/sync_all' endpoint for a Vanta connection (id: ${connectionIdAsString}), an error occurred: ${util.inspect(error.raw)}`);
      }

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
        sails.log.warn('p1: An error occurred while syncing the vanta connection for VantaCustomer with id '+connectionIdAsString+'. Logged error:\n'+errorReportById[connectionIdAsString]);
      }
    }//∞

    sails.log('Information has been sent to Vanta for '+(allActiveVantaConnections.length - numberOfLoggedErrors)+(allActiveVantaConnections.length - numberOfLoggedErrors > 1 || numberOfLoggedErrors === allActiveVantaConnections.length ? ' connections.' : ' connection.'));

  }


};

