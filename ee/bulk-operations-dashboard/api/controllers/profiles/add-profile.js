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


  fn: async function (inputs) {
    // Get filename.
    console.log(inputs);
    // console.log(newProfile);

    // if(teams.length > 0) {
    //   Profile.create({
    //     // name: // filename,
    //     // platform: // Either darwind or windows based on the file extension
    //     profileContents: newProfile,
    //   })
    // } else {
    //   // for(let teamApid in teams){
    //   //   await sails.helpers.http.post.with({
    //   //     baseUrl: sails.config.custom.fleetBaseUrl,
    //   //     url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
    //   //     headers: {
    //   //       Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
    //   //     },
    //   //   });
    //   // }
    // }




    // All done.
    return;

  }


};
