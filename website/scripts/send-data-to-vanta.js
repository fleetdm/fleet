module.exports = {


  friendlyName: 'Send data to vanta',


  description: '',


  fn: async function () {

    const ONE_HOUR_IN_MS = 1000 * 60 * 60;

    // Find all ExternalAuthorization records with a connected status.
    let allVantaConnections = ExternalAuthorization.find({stauts: 'Connected'});


    // use sails.helpers.flow.forEachSimutaniously through each connection
    await sails.helpers.process.flow.simultaneouslyForEach(allVantaConnections, async (vantaConnection)=>{

      // Refresh the authorization token for this connection.

      let newAuthorizationToken = await sails.helpers.http.post(
        'https://api.vanta.com/oauth/token',
        {
          'client_id': sails.config.custom.vantaAuthorizationClientId,
          'client_secret': sails.config.custom.vantaAuthorizationClientSecret,
          'refresh_token': vantaConnection.refreshToken,
          'grant_type': 'refresh_token',
        },
      ).catch(err, (err)=>{
        // TODO: what if this fails?
        // More context: We don't want the script to stop running, because this would handle all Vanta Connections.
      });

      // Update the record in the database.
      let updatedRecord = await ExternalAuthorization.updateOne({id: vantaConnection.id}).with({
        authToken: vantaAuthorizationResponse.access_token,
        authTokenExpiresAt: Date.now() + (vantaAuthorizationResponse.expires_in * 1000),
        refreshToken: vantaAuthorizationResponse.refresh_token,
      });

      // Now send User account information to Vanta.

      let responseFromUserEndpoint = sails.helpers.http.get({
        url: updatedRecord.fleetInstanceUrl + '/api/v1/fleet/users',
        headers: {
          'authorization': 'bearer'+updatedRecord.authToken,
        }
      }).catch(err, (err)=>{
        // TODO, what if a request to the users endpoint fails?
      });

      if(!responseFromUserEndpoint.users){
        // Throw an error if there are no users.
      }

      const vantaPermissionsMappedToFleetGlobalRole = {
        Admin: 'ADMIN',
        Maintainer: 'EDITOR',
        Observer: 'BASE'
      };

      let usersToSendToVanta = [];

      for(let user of responseFromUserEndpoint.users) {

        let authMethod = 'EMAIL';
        if(user.sso_enabled) {
          authMethod = 'SSO';
        }
        let userToSendToVanta = {
          displayName: user.name,
          uniqueId: user.id,
          fullName: user.name,
          accountName: user.name,
          email: user.email,
          permissionLevel: permissionLevels[user.global_role]
          createdTimestamp: user.created_at,
          status: 'ACTIVE',
          mfaEnabled: False,
          mfaMethods: ['DISABLED'],
          authMethod,
        };

        usersToSendToVanta.push(userToSendToVanta);
      }


      // Send a PUT request to Vanta to sync User accounts

      let userSyncResponseFromVanta = await sails.helpers.http.sendHttpResquest({
        method: 'PUT',
        url: 'https://api.vanta.com/v1/resources/user_account/sync_all',
        headers: {
          'accept': 'application/json',
          'authorization': 'Bearer '+updatedRecord.authToken,
          'content-type': 'application/json',
        },
        body: {
          sourceId: externalAuthorizationForThisRequest.sourceID,
          resourceId: ''//TODO: resourceID for user accounts in vanta application
          resources: resourcesForVantaRequest,
        },
      }).catch(err, async (err)=>{
        // TODO, what happens if this fails?
      });



      // Now that we'll start sending requests to the Fleet instance to get information about macOS Hosts.

      let hostsToSendToVanta = sails.helpers.http.get({
        url: updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts',
        headers: {
          'authorization': 'bearer'+updatedRecord.fleetInstanceApiKey,
        }
      }).catch(err, async (err)=>{
        // TODO, what happens if this fails?
      });

      let hostDataForVanta = [];

      await sails.helpers.flow.simultaneouslyForEach(hostsToSendToVanta.hosts, async (host)=>{

        // Start building the host resource to send to Vanta, using information we get from the Fleet instance's get Hosts endpoint
        let macOSHostToSendToVanta = {
          displayName: host.hostname,
          uniqueId: host.id,
          externalUrl: updatedRecord.fleetInstanceUrl + '/hosts/{id}',
          collectedTimestamp: host.updated_at,
          osName: host.os_version.replace(/\s\d\d.\d\d.?\d?\d?$/, ''), // TODO get this a better way
          osVersion: (host.os_version.match(/\d\d.\d\d.?\d?\d?$/)[0] || ''), // TODO get this a better way
          hardwareUuid: host.uuid,
          serialNumber: host.hardware_serial,
          applications: [],
          browserExtensions: [],
          drives: [{
            name: 'drive',
            encrypted: host.disk_encryption_enabled,
            filevaultEnabled: host.disk_encryption_enabled,
          }]
          users: [],// Sending an empty array
          systemScreenlockPolicies: [], // Sending an empty array
          isManaged: false, // Defaulting to false
          autoUpdatesEnabled: false, // Always sending this value as false
        };

        // Send a request to this host's API endpoint to build the arrays of software and brower extensions

        let detailedHostInformation = sails.helpers.http.get({
          url: updatedRecord.fleetInstanceUrl + '/api/v1/fleet/hosts/'+host.id,
          headers: {
            'authorization': 'bearer'+updatedRecord.fleetInstanceApiKey,
          }
        }).catch(err, async (err)=>{
          // TODO, what happens if this fails?
        });

        for(let software of detailedHostInformation.software){
          let softwareToAdd = {
            name: software.name,
          };
          if(software.source === 'firefox_addons'|| software.source === 'chrome_extensions') {
            softwareToAdd.extensionId = software.name + software.version;
            softwareToAdd.browser = software.source.split('_')[0].toUpperCase();
            macOSHostToSendToVanta.applications.push(softwareToAdd);
          } else if(software.source === 'apps'){
            // TODO: filter safari extensions
            softwareToAdd.bundleId = software.bundle_identifier;
            macOSHostToSendToVanta.browserExtensions.push(softwareToAdd);
          }
        }

      });// After every host


      // Now send the array of host resources to Vanta

      let hostSyncResponseFromVanta = await sails.helpers.http.sendHttpResquest({
        method: 'PUT',
        url: 'https://api.vanta.com/v1/resources/user_account/sync_all',
        headers: {
          'accept': 'application/json',
          'authorization': 'Bearer '+updatedRecord.authToken,
          'content-type': 'application/json',
        },
        body: {
          sourceId: externalAuthorizationForThisRequest.sourceID,
          resourceId: '',//TODO: resourceID for hosts in vanta application,
          resources: resourcesForVantaRequest,
        },
      }).catch(err, async (err)=>{
        // TODO, what happens if this fails?
      });


      // Update the dataLastSentToVantaAt timestamp on the record for this host

      await ExternalAuthorization.updateOne({id: updatedRecord.id}).with({
        dataLastSentToVantaAt: Date.now()
      });

    });// After every vanta connection

  }


};

