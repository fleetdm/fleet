module.exports = {


  friendlyName: 'Get sandbox status',


  description: 'Check the status of a user\'s Fleet sandbox instance',


  inputs: {
    // User ID
  },


  exits: {
    // Fleet sandbox could not be found

    // Fleet sandbox is ready
  },


  fn: async function (inputs) {

    // Find user record (User.findOne({id: userID}))
    // If the user does not have a fleetSandboxURL, throw an error

    // Check the /healthz endpoint of the user's fleetSandboxURL until it returns a 200 response
    // await sails.helpers.flow.until(async funtion () {
    //   let serverResponse = sails.helpers.http.get(.....).tolerate('non200Response')
    //   return !! serverResponse
    // });

    // When the Fleet Sandbox instance is ready, return the sandbox url
    // return sandboxURL;

  }


};
