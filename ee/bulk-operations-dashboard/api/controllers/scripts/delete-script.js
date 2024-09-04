module.exports = {


  friendlyName: 'Delete script',


  description: '',


  inputs: {
    script: {
      type: {},
      description: 'The script that will be deleted.',
      required: true,
    }
  },


  exits: {

  },


  fn: async function ({script}) {
    // If the provided script does not have a teams array and has an ID, it is an undeployed script that will be deleted.
    if(script.id && !script.teams){
      await UndeployedScript.destroy({id: script.id});
    } else {
      for(let teamScript of script.teams){
        await sails.helpers.http.sendHttpRequest.with({
          method: 'DELETE',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/scripts/${teamScript.scriptFleetApid}`,
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
