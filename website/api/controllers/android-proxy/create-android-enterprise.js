module.exports = {


  friendlyName: 'Create android enterprise',


  description: 'Creates a new Android enterprise from a request from a Fleet instance.',


  inputs: {
    projectId: {
      type: 'string',
      required: true,
    },
    signupUrlName: {
      type: 'string',
      required: true,
    },
    enterpriseToken: {
      type: 'string',
      required: true,
    },
    fleetLicenseKey: {
      type: 'string',
      // required: true,
    },
    pubsubPushUrl: {
      type: 'string',
      required: true,
    },
    enterprise: {
      type: {},
      required: true,
      moreInfoUrl: 'https://developers.google.com/android/management/reference/rest/v1/enterprises'
    }
  },


  exits: {
    success: { description: 'An android enterprise was successfully created' },
    enterpriseAlreadyExists: { description: 'An android enterprise already exists for this Fleet instance.', responseType: 'badRequest' },
  },


  fn: async function ({projectId, signupUrlName, enterpriseToken, fleetLicenseKey, pubsubPushUrl, enterprise}) {


    // Parse the fleet server url from the origin header.
    let fleetServerUrl = this.req.get('Origin');
    if(!fleetServerUrl){
      return this.res.badRequest();
    }

    // Check the databse for a record of this enterprise.
    let connectionforThisInstanceExists = await AndroidEnterprise.findOne({pubsubPushUrl: pubsubPushUrl});


    if(connectionforThisInstanceExists){
      throw 'enterpriseAlreadyExists';
    }

    // Get an accesn token for the requests to the Android management API
    let authorizationTokenForThisRequest = await sails.helpers.androidEnterprise.getAccessToken.with({
      // TODO: this helper doesn't exist
    });


    let newFleetServerSecret = await sails.helpers.strings.random.with({len: 30});


    let createEnterpriseResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `https://androidmanagement.googleapis.com/v1/enterprises?projectId=${encodeURIComponent(projectId)}&signupUrlName=${encodeURIComponent(signupUrlName)}&enterpriseToken=${enterpriseToken}`,
      data: enterprise,// TODO: Is this how google's API expects this?, or will it need to be { enterprise: newEnterprise }
      headers: {
        Authorization: `Bearer ${authorizationTokenForThisRequest}`
      },
    }).intercept((error)=>{
      return new Error(`An error occured when sending a request to create a new Android enterprise. Full error: ${require('util').inpsect(error)}`);
    });


    // TODO: find a real response from this endpoint so we know what to actually set.
    let newAndroidEnterpriseId = createEnterpriseResponse.id;

    // Create a databse record for the newly created enterprise
    await AndroidEnterprise.createOne({
      fleetServerUrl: fleetServerUrl,
      fleetLicenseKey: fleetLicenseKey,
      fleetServerSecret: newFleetServerSecret,
      androidEnterpriseId: newAndroidEnterpriseId
    });


    return {
      android_enterprise_id: newAndroidEnterpriseId,// eslint-disable-line camelcase
    };

  }


};
