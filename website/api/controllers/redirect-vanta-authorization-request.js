module.exports = {


  friendlyName: 'Redirect vanta authorization request',


  description: 'Sets provided inputs in the user`s browser as cookies and redirects them to Vanta.',


  inputs: {
    vantaSourceId: {
      type: 'string',
      description: 'The generated vanta Source ID for this request.',
      required: true,
    },
    state: {
      type: 'string',
      description: 'The state provided to Vanta when an authorization request was created',
      required: true,
    },
    vantaAuthorizationRequestURL: {
      type: 'string',
      description: 'The Vanta authorization url that the user will be directed to after they are sent to this page.',
      required: true,
    },
    redirectAfterSetup: {
      type: 'string',
      description: 'The URL that the user will be redirected to after they complete setup.',
      required: true,
    }
  },


  exits: {
    noMatchingVantaConnection: {
      description: 'No Vanta connection could be found using the provided vantaSourceId',
      responseType: 'badRequest'
    },
  },


  fn: async function ({vantaSourceId, state, vantaAuthorizationRequestURL, redirectAfterSetup}) {

    // Find the VantaConnection record that we created when the user created this request.
    let recordOfThisAuthorization = await VantaConnection.findOne({vantaSourceId: vantaSourceId});

    // If no record of this authorization could be found, return a noMatchingVantaConnection response.
    if(!recordOfThisAuthorization){
      throw 'noMatchingVantaConnection';
    }

    // Set a 'state' and 'vantaSourceId' cookie on the users browser.
    this.res.cookie('redirectAfterSetup', redirectAfterSetup, {signed: true});
    this.res.cookie('state', state, {signed: true});
    this.res.cookie('vantaSourceId', vantaSourceId, {signed: true});
    // now that the user has the required cookies to complete the vanta integration setup, redirect them to the provided VantaAuthorizationUrl.
    return this.res.redirect(vantaAuthorizationRequestURL);
  }


};
