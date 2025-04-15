module.exports = {


  friendlyName: 'Delete one android enterprise',


  description: 'Deletes an android enterprise and the associated database record.',


  inputs: {
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'An Android enterprise was successfully deleted.' }
  },


  fn: async function ({fleetServerSecret, androidEnterpriseId}) {

    // Look up the database record for this Android enterprise
    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      fleetServerSecret: fleetServerSecret,
      androidEnterpriseId: androidEnterpriseId,
    });

    // Return a 404 response if no records are found.
    if(!thisAndroidEnterprise) {
      return this.res.notFound();
    }
    // Delete the Android enterprise
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    await sails.helpers.flow.build(async ()=>{
      let google = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');
      let googleAuth = new google.auth.GoogleAuth({
        scopes: [
          'https://www.googleapis.com/auth/androidmanagement',
          'https://www.googleapis.com/auth/pubsub'
        ],
        credentials: {
          client_email: sails.config.custom.GoogleClientId,// eslint-disable-line camelcase
          private_key: sails.config.custom.GooglePrivateKey,// eslint-disable-line camelcase
        },
      });
      // Acquire the google auth client, and bind it to all future calls
      let authClient = await googleAuth.getClient();
      google.options({auth: authClient});
      // Delete the android enterprise.
      await androidmanagement.enterprises.delete({
        name: `enterprises/${androidEnterpriseId}`,
      });
      let pubsub = google.pubsub('v1');
      // Delete the enterprise's pubsub topic
      await pubsub.projects.topics.delete({
        topic: thisAndroidEnterprise.pubsubTopicName,
      });
      return;
    }).intercept((err)=>{
      return new Error(`When attempting to delete an android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });

    // Delete the database record for this Android enterprise
    await AndroidEnterprise.destroyOne({ id: thisAndroidEnterprise.id });


    // All done.
    return;

  }


};
