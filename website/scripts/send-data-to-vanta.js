module.exports = {


  friendlyName: 'Send data to vanta',


  description: '',


  fn: async function () {
    process.env["NODE_TLS_REJECT_UNAUTHORIZED"] = 0; // Remove, used for testing https requests to local Fleet instance.
    const ONE_HOUR_IN_MS = 1000 * 60 * 60;
    let allToleratedErrors = [];
    // Find all VantaConnection records with a connected status.
    let allVantaConnections = await VantaConnection.find({isConnectedToVanta: true});

    // use sails.helpers.flow.forEachSimutaniously to send data for each connection record.
    await sails.helpers.flow.simultaneouslyForEach(allVantaConnections, async (vantaConnection)=>{

      // Refresh the authorization token for this connection.

      // let newAuthorizationToken = await sails.helpers.http.post(
      //   'https://api.vanta.com/oauth/token',
      //   {
      //     'client_id': sails.config.custom.vantaAuthorizationClientId,
      //     'client_secret': sails.config.custom.vantaAuthorizationClientSecret,
      //     'refresh_token': vantaConnection.refreshToken,
      //     'grant_type': 'refresh_token',
      //   },
      // ).catch(async (err)=>{
      //   sails.log(`When refreshing the authorization token for a Vanta connection (id: ${vantaConnection.id}), Vanta returned an error. This user will need to manually reconnect Vanta to continue. Check the error logged on the database record for more information.`)
      //   return await VantaConnection.updateOne({id: vantaConnection.id}).set({
      //     isConnectedToVanta: false;
      //     lastErrorAboutThisConnection: JSON.stringify(err);
      //   });
      // });

      // Update the record in the database.
      let updatedRecord = await VantaConnection.updateOne({ id: vantaConnection.id }).set({
        authTokenExpiresAt: Date.now()
      });

      let continueScriptForThisRecord = true;

      //  ┬ ┬┌─┐┌─┐┬─┐┌─┐
      //  │ │└─┐├┤ ├┬┘└─┐
      //  └─┘└─┘└─┘┴└─└─┘
      // Request user data from the Fleet instance to send to Vanta.
      sails.log('sending request for '+vantaConnection.id);
      let responseFromUserEndpoint = await sails.helpers.http.get(
        updatedRecord.fleetInstanceUrl + '/api/v1/fleet/users',
        {},
        {'Authorization': 'Bearer '+updatedRecord.fleetApiKey }
      ).retry()
      .catch(async (err)=>{
        continueScriptForThisRecord = false;
        return new Error(`When sending a request to the Fleet instance's /users endpoint for a Vanta connection (id: ${vantaConnection.id}), an error occurred.`);
      });

      if(!continueScriptForThisRecord){
        await VantaConnection.updateOne({id: vantaConnection.id}).set({
          isConnectedToVanta: false,
          lastErrorAboutThisConnection: {
            errorDescription: 'An Error occured while sending a request to the /users endpoint of the Fleet instance.',
            fullError: responseFromUserEndpoint
          },
        });
        allToleratedErrors.push({recordId: vantaConnection.id, error: 'An Error occured while sending a request to the /users endpoint of the Fleet instance.'});
        return;
      }

      // using an object with the keys set to Fleet's global roles, we can use the value from the Fleet instance to pick the coresponding Vanta role.
      const vantaPermissionsMappedToFleetGlobalRole = {
        Admin: 'ADMIN',
        Maintainer: 'EDITOR',
        Observer: 'BASE'
      };

      let usersToSendToVanta = [];

      // Iterate through the users list, creating user objects to send to Vanta.
      for(let user of responseFromUserEndpoint.users) {

        // Default the user's authMethod to `EMAIL`, If the user has sso_enabled: true, we'll set it to SSO
        let authMethod = 'EMAIL';
        if(user.sso_enabled) {
          authMethod = 'SSO';
        }
        // Create a user object for this Fleet user.
        let userToSendToVanta = {
          displayName: user.name,
          uniqueId: user.id,
          fullName: user.name,
          accountName: user.name,
          email: user.email,
          permissionLevel: vantaPermissionsMappedToFleetGlobalRole[user.global_role],
          createdTimestamp: user.created_at,
          status: 'ACTIVE',
          mfaEnabled: false,
          mfaMethods: ['DISABLED'],
          authMethod,
        };

        // Add the user object to the array of users for Vanta.
        usersToSendToVanta.push(userToSendToVanta);
      }


      //  ┬ ┬┌─┐┌─┐┌┬┐┌─┐
      //  ├─┤│ │└─┐ │ └─┐
      //  ┴ ┴└─┘└─┘ ┴ └─┘
      // Now that we'll start sending requests to the Fleet instance to get information about hosts.
      let responseFromHostsEndpoint = await sails.helpers.http.get(
        updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts',
        {},
        {'Authorization': 'bearer '+updatedRecord.fleetApiKey},
      ).retry().catch(async (err)=>{
        continueScriptForThisRecord = false;
        return new Error(`fleetInstanceError: When sending a request to the Fleet instance's /hosts endpoint for a Vanta connection (id: ${vantaConnection.id}), an error occurred.`);
      });

      if(!continueScriptForThisRecord){
        await VantaConnection.updateOne({id: vantaConnection.id}).set({
          isConnectedToVanta: false,
          lastErrorAboutThisConnection: {
            errorDescription: 'An Error occured while sending a request to the /hosts endpoint of the Fleet instance for '+vantaConnection.id,
            fullError: allHostsOnThisFleetInstance
          }
        });
        allToleratedErrors.push({recordId: vantaConnection.id, error: 'An Error occured while sending a request to the /hosts endpoint of the Fleet instance'});
        return;
      }

      // If there is a maximum number of hosts returned in /hosts api request
      // let pageNumberForPossiblePaginatedResults = 0;
      // let allHostsOnThisFleetInstance = [];
      // await sails.helpers.flow.until(async ()=>{
      // let hostsToSendToVanta = await sails.helpers.http.get(
      //   updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts?per_page=100&page='+pageNumberForPossiblePaginatedResults,
      //   {},
      //   {
      //     'Authorization': 'bearer '+updatedRecord.fleetApiKey,
      //   },
      // ).retry()
      //   // Add the results to the allHostsOnThisFleetInstance array.
      //   allHostsOnThisFleetInstance = allHostsOnThisFleetInstance.concat(hostsToSendToVanta.hosts);
      //   // Increment the page of results we're requesting.
      //   pageNumberForPossiblePaginatedResults += 1;
      //   // If we recieved less results than we requested, we've reached the last page of the results.
      //   return hostsToSendToVanta.length !== 100;
      // }, 10000);


      let macOsHosts = responseFromHostsEndpoint.hosts.filter((host)=>{
        return host.platform === 'darwin';
      })

      let hostDataForVanta = [];

      await sails.helpers.flow.simultaneouslyForEach(macOsHosts, async (host)=>{

        // Start building the host resource to send to Vanta, using information we get from the Fleet instance's get Hosts endpoint
        let macOSHostToSendToVanta = {
          displayName: host.display_name,
          uniqueId: host.id,
          externalUrl: updatedRecord.fleetInstanceUrl + '/hosts/'+host.id,
          collectedTimestamp: host.updated_at,
          osName: 'macOS', // Defaulting this value to macOS
          osVersion: host.os_version.replace(/^([\D]+)\s(.+)/g, '$2'),
          hardwareUuid: host.uuid,
          serialNumber: host.hardware_serial,
          applications: [],
          browserExtensions: [],
          drives: [{
            name: 'drive',
            encrypted: host.disk_encryption_enabled,
            filevaultEnabled: host.disk_encryption_enabled,
          }],
          users: [],// Sending an empty array
          systemScreenlockPolicies: [], // Sending an empty array
          isManaged: false, // Defaulting to false
          autoUpdatesEnabled: false, // Always sending this value as false
        };


        // Send a request to this host's API endpoint to build the arrays of software and browser extensions
        let detailedInformationAboutThisHost = await sails.helpers.http.get(
          updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts/'+host.id,
          {},
          {'Authorization': 'bearer '+updatedRecord.fleetApiKey}
        )
        .retry()
        .catch(async (err)=>{
          await VantaConnection.updateOne({id: vantaConnection.id}).set({
            isConnectedToVanta: false,
            lastErrorAboutThisConnection: {
              errorDescription: 'An Error occured while sending a request to the hosts/{id} endpoint of the Fleet instance.',
              fullError: err
            },
          });
          throw new Error(`fleetInstanceError: When sending a request to the Fleet instance's /hosts/{id} endpoint for a Vanta connection (id: ${vantaConnection.id}), an error occurred.`);
        });

        if(!continueScriptForThisRecord) {
          await VantaConnection.updateOne({id: vantaConnection.id}).set({
            isConnectedToVanta: false,
            lastErrorAboutThisConnection: {
              errorDescription: 'An Error occured while sending a request to the hosts/{id} endpoint of the Fleet instance for a Vanta connection'+vantaConnection.id,
              fullError: allHostsOnThisFleetInstance
            }
          });

          return;
        }

        if(!detailedInformationAboutThisHost.host.software) {
          throw new Error('A host is missing an array of software.');
        }

        for(let software of detailedInformationAboutThisHost.host.software){
          let softwareToAdd = {
            name: software.name,
          };
          if(software.source === 'firefox_addons'|| software.source === 'chrome_extensions') {
            softwareToAdd.extensionId = software.name + software.version;
            softwareToAdd.browser = software.source.split('_')[0].toUpperCase();
            macOSHostToSendToVanta.browserExtensions.push(softwareToAdd);
          } else if(software.source === 'apps' && software.bundle_identifier){
            // TODO: filter safari extensions
            softwareToAdd.bundleId = software.bundle_identifier;
            macOSHostToSendToVanta.applications.push(softwareToAdd);
          }
        }
        hostDataForVanta.push(macOSHostToSendToVanta);
      });// After every host


      //  ┌─┐┌─┐┌┐┌┌┬┐  ┌┬┐┌─┐┌┬┐┌─┐  ┌┬┐┌─┐  ┬  ┬┌─┐┌┐┌┌┬┐┌─┐
      //  └─┐├┤ │││ ││   ││├─┤ │ ├─┤   │ │ │  └┐┌┘├─┤│││ │ ├─┤
      //  └─┘└─┘┘└┘─┴┘  ─┴┘┴ ┴ ┴ ┴ ┴   ┴ └─┘   └┘ ┴ ┴┘└┘ ┴ ┴ ┴



      // Send a PUT request to Vanta to sync User accounts
      // let userSyncResponseFromVanta = await sails.helpers.http.sendHttpResquest({
      //   method: 'PUT',
      //   url: 'https://api.vanta.com/v1/resources/user_account/sync_all',
      //   body: {
      //     sourceId: vantaConnection.emailAddress,
      //     resourceId: '63868a88d436911435a94035',
      //     resources: usersToSendToVanta,
      //   },
      //   headers: {
      //     'accept': 'application/json',
      //     'authorization': 'Bearer '+updatedRecord.authToken,
      //     'content-type': 'application/json',
      //   },
      // }).catch(async (err)=>{
      //   // TODO, what happens if this fails?
      // });

      // Send a PUT request to Vanta to sync macOS hosts
      // let hostSyncResponseFromVanta = await sails.helpers.http.sendHttpResquest({
      //   method: 'PUT',
      //   url: 'https://api.vanta.com/v1/resources/user_account/sync_all',
      //   body: {
      //     sourceId: vantaConnectionForThisRequest.sourceID,
      //     resourceId: '',//TODO: resourceID for hosts in vanta application,
      //     resources: hostDataForVanta,
      //   },
      //   headers: {
      //     'accept': 'application/json',
      //     'authorization': 'Bearer '+updatedRecord.authToken,
      //     'content-type': 'application/json',
      //   },
      // }).catch(async (err)=>{
      //   await VantaConnection.updateOne({id: vantaConnection.id}).set({
      //     isConnectedToVanta: false,
      //     lastErrorAboutThisConnection: JSON.stringify(err),
      //   });
      //   throw new Error(`Could not send host data to Vanta (id: ${vantaConnection.id}).`);
      // });


      // Testing output, TODO: delete
      sails.log('Data that would have been sent to Vanta for connection will be output to a json file!');
      // sails.log(usersToSendToVanta);
      // sails.log(hostDataForVanta);
      let requestToVanta = {
        'hostResources': hostDataForVanta,
        'usersToSendToVanta': usersToSendToVanta
      };
      await sails.helpers.fs.writeJson(sails.config.appPath + '/'+vantaConnection.id+'-vanta-test.json', requestToVanta, true);


      // Update the dataLastSentToVantaAt timestamp on the record for this host

      // await VantaConnection.updateOne({id: updatedRecord.id}).with({
      //   dataLastSentToVantaAt: Date.now()
      // });

    }).tolerate((error)=>{
      // Because we're syncing data for all vanta connections, we need this script to continue running, even if an error occurs on a single host.

    });// After every vanta connection
    sails.log('This script is done... '+allToleratedErrors.length+' total errors',allToleratedErrors);

  }


};

