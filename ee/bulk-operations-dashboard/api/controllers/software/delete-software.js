module.exports = {


  friendlyName: 'Delete software',


  description: 'Deletes deployed software for all teams on a Fleet instance, or undeployed software in the app\'s database',

  inputs: {
    software: {
      type: {},
      description: 'The software that will be deleted.',
      required: true,
    }
  },


  exits: {
    softwareDeletionFailed: {
      description: 'The specified software could not be deleted from the Fleet instance.',
      statusCode: 409,
    }
  },


  fn: async function ({software}) {
    // If the provided software does not have a teams array and has an ID, it is an undeployed software that will be deleted.
    if(software.id && !software.teams){
      await sails.rm(sails.config.uploads.prefixForFileDeletion+software.uploadFd);
      await UndeployedSoftware.destroy({id: software.id});
    } else {// Otherwise, this is a deployed software, and we'll use information from the teams array to remove the software.
      for(let team of software.teams){
        await sails.helpers.http.sendHttpRequest.with({
          method: 'DELETE',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/software/titles/${software.fleetApid}/available_for_install?team_id=${team.fleetApid}`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        })
        .intercept({raw:{statusCode: 409}}, (error)=>{
          // If the Fleet instance's returns a 409 response, then the software is configured to be installed as
          // part of the macOS setup experience, and must be removed before it can be deleted via API requests.
          return {softwareDeletionFailed: error};
        });
      }
    }
    // All done.
    return;

  }


};
