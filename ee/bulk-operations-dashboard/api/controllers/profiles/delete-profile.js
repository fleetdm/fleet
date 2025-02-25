module.exports = {


  friendlyName: 'Delete profile',


  description: '',


  inputs: {
    profile: {
      type: {},
      description: 'The configuration profile that will be deleted.',
      required: true,
    }
  },


  exits: {

  },


  fn: async function ({profile}) {
    // If the provided profile does not have a teams array and has an ID, it is an undeployed profile that will be deleted.
    if(profile.id && !profile.teams){
      await UndeployedProfile.destroy({id: profile.id});
    } else {// Otherwise, this is a deployed profile, and we'll use information from the teams array to remove the profile.
      for(let team of profile.teams){
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
    // All done.
    return;

  }


};
