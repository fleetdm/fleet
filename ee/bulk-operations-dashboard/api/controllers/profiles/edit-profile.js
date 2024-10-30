module.exports = {


  friendlyName: 'Edit profile',


  description: 'Edits the teams a profile is assigned to and/or replaces the file on the Fleet instance if the new file\'s profile identifier matches',

  files: ['newProfile'],

  inputs: {
    profile: {
      type: {},
      description: 'The configuration profile that is being editted',
      required: true,
    },
    newTeamIds: {
      type: ['string'],
      description: 'An array of teams that this profile will be deployed on or Undefined if the profile is being removed from a team.'
    },
    newProfile: {
      type: 'ref',
      description: 'A file that will be replacing the profile.'
    },
    profileTarget: {
      type: 'string',
      description: 'The target for this configuration profile',
      defaultsTo: 'all',
      isIn: ['all', 'custom'],
    },
    labelTargetBehavior: {
      type: 'string',
      isIn: ['include', 'exclude'],
    },
    labels: {
      type: ['string'],
      description: 'A list of the names of labels that will be included/excluded.'
    }
  },


  exits: {
    payloadIdentifierDoesNotMatch: {
      statusCode: 409,
      description: 'The new profiles bundle indentifer does not match the existing profile',
    }
  },



  fn: async function ({profile, newTeamIds, newProfile, profileTarget, labelTargetBehavior, labels}) {
    if(newProfile.isNoop){
      newProfile.noMoreFiles();
      newProfile = undefined;
    }
    //  ╔═╗╔═╗╔╦╗  ╔═╗╦═╗╔═╗╔═╗╦╦  ╔═╗
    //  ║ ╦║╣  ║   ╠═╝╠╦╝║ ║╠╣ ║║  ║╣
    //  ╚═╝╚═╝ ╩   ╩  ╩╚═╚═╝╚  ╩╩═╝╚═╝
    let profileContents; // The raw text contents of a profile file.
    let filename;
    let extension;
    // If there is not a new profile, and the profile is deployed (has teams array === deployed), download the profile to be able to add it to other teams.
    if(!newProfile && profile.teams){
      // console.log('Existing deployed profile!');
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
    } else if(newProfile) {// Otherwise, if there is a new profile file uploaded, check that the payload identifier maches the existing profile on the Fleet instance.
      // console.log('Replacing an existing(/undeployed) profile!');
      let file = await sails.reservoir(newProfile);
      profileContents = file[0].contentBytes;
      let profileFileName = file[0].name;
      filename = profileFileName.replace(/^\d{4}-\d{2}-\d{2}_/, '').replace(/\.[^/.]+$/, '');
      extension = '.'+profileFileName.split('.').pop();
      let profilePlatform = 'darwin';
      if(_.endsWith(profileFileName, '.xml')) {
        profilePlatform = 'windows';
      }
      if(newTeamIds && profile.teams && profilePlatform === 'darwin'){
        let existingProfileInfo = await sails.helpers.http.get.with({
          url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/configuration_profiles/${profile.teams[0].uuid}?alt=media`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
          }
        });
        let newProfileBundleIdentifier = profileContents.match(/<key>PayloadIdentifier<\/key>\s*<string>(.*?)<\/string>/)[1];
        let existingProfileBundleIdentifier = existingProfileInfo.match(/<key>PayloadIdentifier<\/key>\s*<string>(.*?)<\/string>/)[1];
        // Note: We're using the _.startsWith method to check that the identifier is the same. The identifiers returned by the Fleet instance are
        if(existingProfileBundleIdentifier !== newProfileBundleIdentifier){
          throw 'payloadIdentifierDoesNotMatch';
        }
      }
    } else if (!newProfile && !profile.teams){// Undeployed profiles are stored in the app's database.
      // console.log('editing an undeployed profile!');
      profileContents = profile.profileContents;
      filename = profile.name;
      extension = profile.profileType;
    }


    //  ╔═╗╔═╗╔═╗╦╔═╗╔╗╔  ╔═╗╦═╗╔═╗╔═╗╦╦  ╔═╗
    //  ╠═╣╚═╗╚═╗║║ ╦║║║  ╠═╝╠╦╝║ ║╠╣ ║║  ║╣
    //  ╩ ╩╚═╝╚═╝╩╚═╝╝╚╝  ╩  ╩╚═╚═╝╚  ╩╩═╝╚═╝
    if(!newProfile){
      // If we're changing the teams for an existing profile, we'll remove this profile from any team not included in the newTeamIds array.
      let currentProfileTeamIds = _.pluck(profile.teams, 'fleetApid');
      let addedTeams = _.difference(newTeamIds, currentProfileTeamIds);
      let removedTeams = _.difference(currentProfileTeamIds, newTeamIds);
      let removedTeamsInfo = _.filter(profile.teams, (team)=>{
        return removedTeams.includes(team.fleetApid);
      });
      for(let team of removedTeamsInfo){
        // console.log(`removing ${profile.name} from team id ${team.teamName}`);
        await sails.helpers.http.sendHttpRequest.with({
          method: 'DELETE',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles/${team.uuid}`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
      }
      for(let teamApid of addedTeams){
        // console.log(`Adding ${profile.name} to team id ${teamApid}`);
        let bodyForThisRequest = {
          team_id: teamApid,// eslint-disable-line camelcase
          labels_exclude_any: labelTargetBehavior === 'exclude' ? labels : undefined,// eslint-disable-line camelcase
          labels_include_all: labelTargetBehavior === 'include' ? labels : undefined,// eslint-disable-line camelcase
          profile: {
            options: {
              filename: filename + extension,
              contentType: 'application/octet-stream'
            },
            value: profileContents,
          }
        };
        await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
          enctype: 'multipart/form-data',
          body: bodyForThisRequest,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          },
        });
      }// After every added team
    } else {
      if(profile.teams) {
        // If there is a new profile uploaded, we will need to delete the old profiles, and add the new profile.
        for(let team of profile.teams) {
          // console.log(`removing ${profile.name} from team id ${team.teamName}`);
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            baseUrl: sails.config.custom.fleetBaseUrl,
            url: `/api/v1/fleet/configuration_profiles/${team.uuid}`,
            headers: {
              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
            }
          });
        }
      }
      for(let teamApid of newTeamIds){
        let bodyForThisRequest = {
          team_id: teamApid,// eslint-disable-line camelcase
          labels_exclude_any: labelTargetBehavior === 'exclude' ? labels : undefined,// eslint-disable-line camelcase
          labels_include_all: labelTargetBehavior === 'include' ? labels : undefined,// eslint-disable-line camelcase
          profile: {
            options: {
              filename: filename + extension,
              contentType: 'application/octet-stream'
            },
            value: profileContents,
          }
        };
        // console.log(`Adding ${profile.name} to team id ${teamApid}`);
        await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
          enctype: 'multipart/form-data',
          body: bodyForThisRequest,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          },
        });
      }// After every added team

    }
    // If this profile has an ID, then it is a database record, and we will delete it if it has been deployed to a team.
    if(profile.id && newTeamIds.length > 0){
      // console.log('Undeployed profile has been deployed. deleting DB record!');
      await UndeployedProfile.destroy({id: profile.id});
    } else if(!profile.id && newTeamIds.length === 0){
      // If this is not a database record of a profile, and the profile is being undeployed from all teams, we'll create a databse record for it.
      // console.log('Creating database record for a (now) undeployed profile!');
      await UndeployedProfile.create({
        name: profile.name,
        platform: extension === '.xml' ? 'windows' : 'darwin',
        profileContents,
        profileType: extension,
        labels,
        labelTargetBehavior,
        profileTarget,
      });
    } else if(profile.id && newProfile){
      // If there is a new profile that is replacing a database record, update the profileContents in the database.
      // console.log('Updating existing undeployed profile!');
      await UndeployedProfile.updateOne({id: profile.id}).set({
        profileContents,
        labels,
        labelTargetBehavior,
        profileTarget,
      });
    } else if(profile.id && labels) {
      // Update label target behavior for undeployed profiles.
      await UndeployedProfile.updateOne({id: profile.id}).set({
        profileContents,
        labels,
        labelTargetBehavior,
        profileTarget,
      });
    }
    // All done.
    return;

  }


};
