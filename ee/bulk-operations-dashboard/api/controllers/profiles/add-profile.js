module.exports = {


  friendlyName: 'Add profile',


  description: '',

  files: ['newProfile'],

  inputs: {
    newProfile: {
      type: 'ref',
      description: 'An Upstream with an incoming file upload.',
      // required: true,
    },
    teams: {
      type: ['ref'],
      description: 'An array of team IDs that this profile will be added to'
    }
  },


  exits: {

  },


  fn: async function ({newProfile, teams}) {
    // console.log(newProfile);

    let path = require('path');
    let fileStream = newProfile._files[0].stream;
    let profileFileName = fileStream.filename;
    let datelessExtensionlessFilename = profileFileName.replace(/^\d{4}-\d{2}-\d{2}_/, '').replace(/\.[^/.]+$/, '');
    let extension = '.'+profileFileName.split('.').pop();
    let tempFilePath = `.tmp/${profileFileName}`;
    let profilePlatform = 'darwin';
    if(_.endsWith(profileFileName, '.xml')) {
      profilePlatform = 'windows';
    }
    await sails.helpers.fs.writeStream.with({
      sourceStream: newProfile._files[0].stream,
      destination: tempFilePath,
      force: true,
    });
    let profileContents = await sails.helpers.fs.read(tempFilePath);
    await sails.helpers.fs.rmrf(path.join(sails.config.appPath, tempFilePath));


    let profileToReturn;
    let newProfileInfo = {
      name: datelessExtensionlessFilename,
      platform: _.endsWith(profileFileName, '.xml') ? 'windows' : 'darwin',
      profileType: extension,
      createdAt: Date.now()
    };
    if(!teams) {
      newProfileInfo.profileContents = profileContents;
      profileToReturn = await UndeployedProfile.create(newProfileInfo).fetch();
    } else {
      let newTeams = [];
      for(let teamApid of teams){
        let newProfileResponse = await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
          enctype: 'multipart/form-data',
          body: {
            team_id: teamApid,
            profile: {
              options: {
                filename: profileFileName,
                contentType: 'application/octet-stream'
              },
              value: profileContents,
            }
          },
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          },
        });
        let parsedJsonResponse = JSON.parse(newProfileResponse.body)
        let uuidForThisProfile = parsedJsonResponse.profile_uuid;
        // send a request to the Fleet instance to get the bundleId of the new profile.
        await sails.helpers.http.get.with({
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles/${uuidForThisProfile}`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
        newTeams.push({
          fleetApid: teamApid,
          uuid: JSON.parse(newProfileResponse.body).profile_uuid
        })
      }
      newProfileInfo.teams = newTeams;
      profileToReturn = newProfileInfo;
    }




    // All done.
    return profileToReturn;

  }


};
