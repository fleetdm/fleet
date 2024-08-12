module.exports = {


  friendlyName: 'Edit profile',


  description: '',


  inputs: {
    profile: {
      type: {},
      description: 'The configuration profile that is being editted',
      required: true,
    },
    newTeamIds: {
      type: ['ref'],
      description: 'An array of teams that this profile will be deployed on or Undefined if the profile is being removed from a team.'
    },
    newProfile: {
      type: 'ref',
      description: 'A file that will be replacing the profile.'
    },
  },


  exits: {

  },
  // For Monday eric:
  // What this does currently:
  // Downlaods a profile if a new one is not provided.

  // Flows to add:
  // Replacing a configuration file.
  //  - with new teams
  //  - without new teams
  // Replacing a deployed configuration file.
  // reaplacing an deployed profile and removing the teams
  //
  fn: async function ({profile, newTeamIds, newProfile}) {
    console.log(profile);
    console.log('newTeamIds', newTeamIds)
    console.log('newProfile', newProfile)

    let profileContents; // The raw text contents of a profile file.
    let filename;
    let extension;
    // If there is not a new profile, and the profile is deployed (has teams array === deployed), download the profile to be able to add it to other teams.
    if(!newProfile && profile.teams){
      let profileUuid = profile.teams[0].uuid;
      let profileDownloadResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'GET',
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/configuration_profiles/${profileUuid}?alt=media`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      });
      let contentDispositionHeader = profileDownloadResponse.headers['content-disposition'];
      let filenameMatch = contentDispositionHeader.match(/filename="(.+?)"/);
      filename = filenameMatch[1];
      extension = '.'+filename.split('.').pop();
      profileContents = profileDownloadResponse.body;
    } else if(newProfile) {
      let path = require('path');
      let fileStream = newProfile._files[0].stream;
      let profileFileName = fileStream.filename;
      filename = profileFileName.replace(/^\d{4}-\d{2}-\d{2}_/, '').replace(/\.[^/.]+$/, '');
      extension = '.'+profileFileName.split('.').pop();
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
      profileContents = await sails.helpers.fs.read(tempFilePath);
      await sails.helpers.fs.rmrf(path.join(sails.config.appPath, tempFilePath));
    } else if (!newProfile && !profile.teams){// Undeployed profiles are stored in the app's database.
      profileContents = profile.profileContents;
      filename = profile.name + profile.profileType;
      extension = profile.profileType;
    }






    // If this is a deployed profile, get a list of teams that have been added, and teams that have been removed.
    // if(profile.teams) {
      // Note: there might be an edgecase where we don't have this information. If a profile is added, the uuid is added to the profiles object until a page refresh.
      // ∆: update add profile endpint to send a request to get the added profile's uuid to prevent this.

      let currentProfileTeamIds = _.pluck(profile.teams, 'fleetApid');
      let addedTeams = _.difference(newTeamIds, currentProfileTeamIds);
      let removedTeams = _.difference(currentProfileTeamIds, newTeamIds);
      console.log('added!', addedTeams);
      console.log('removed!',removedTeams);

      let removedTeamsInfo = _.filter(profile.teams, (team)=>{
        return removedTeams.includes(team.fleetApid);
      });
      console.log(removedTeamsInfo);
      for(let team of removedTeamsInfo){
        console.log(`removing ${profile.name} from team id ${team.name}`);
        await sails.helpers.http.sendHttpRequest.with({
          method: 'DELETE',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles/${team.uuid}`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        })
      }

      let newTeams = [];
      for(let teamApid of addedTeams){
        console.log(`adding ${profile.name} to team id ${teamApid}`);
        let newProfileResponse = await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
          enctype: 'multipart/form-data',
          body: {
            team_id: teamApid,
            profile: {
              options: {
                filename: filename,
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
        newTeams.push({
          fleetApid: teamApid,
          uuid: JSON.parse(newProfileResponse.body).profile_uuid
        })
      }
      if(profile.id){
        await UndeployedProfile.destroy({id: profile.id});
      }

    // }
      if(profile.teams && !profile.id && newTeamIds.length > 0) {
      await UndeployedProfile.create({
        name: filename,
        platform: extension === '.xml' ? 'windows' : 'darwin',
        profileContents,
        profileType: extension,
      });
    } else if (profile.id){
      await UndeployedProfile.updateOne({id: profile.id}).set({
        profileContents,
      });
    }







    // All done.
    return;

  }


};
