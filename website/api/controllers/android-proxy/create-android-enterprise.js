module.exports = {


  friendlyName: 'Create android enterprise',


  description: 'Creates a new Android enterprise from a request from a Fleet instance.',


  inputs: {
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
    },
    pubsubPushUrl: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'An android enterprise was successfully created' },
    enterpriseAlreadyExists: { description: 'An android enterprise already exists for this Fleet instance.', statusCode: 409 },
  },


  fn: async function ({signupUrlName, enterpriseToken, fleetLicenseKey, pubsubPushUrl, fleetServerSecret}) {


    // Check the database for a record of this enterprise.
    let connectionforThisInstanceExists = await AndroidEnterprise.findOne({fleetServerSecret: fleetServerSecret});
    if(!connectionforThisInstanceExists) {
      throw this.res.notFound();
    }
    // If this request came from a Fleet instance that already has an enterprise set up, return an error.
    if(connectionforThisInstanceExists.androidEnterpriseId) {
      throw 'enterpriseAlreadyExists';
    }
    // Generate a uuid to use for the pubsub topic name for this Android enterprise.
    let newPubSubTopicName = 'a' + sails.helpers.strings.uuid();// Google requires that topic names start with a letter, so we'll preprend an 'a' to the generated uuid.
    // Build the full pubsub topic name.
    let fullPubSubTopicName = `projects/${sails.config.custom.androidManagementProjectId}/topics/${newPubSubTopicName}`;

    // Complete the setup of the new Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let newEnterprise = await sails.helpers.flow.build(async ()=>{
      // [?] https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Signupurls.html#create
      let google = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');

      let googleAuth = new google.auth.GoogleAuth({
        scopes: [
          'https://www.googleapis.com/auth/pubsub',// For creating the PubSub topic
          'https://www.googleapis.com/auth/androidmanagement'// For creating the new Android enterprise
        ],
        credentials: {
          client_email: sails.config.custom.GoogleClientId,// eslint-disable-line camelcase
          private_key: sails.config.custom.GooglePrivateKey,// eslint-disable-line camelcase
        },
      });
      // Acquire the google auth client, and bind it to all future calls
      let authClient = await googleAuth.getClient();
      google.options({auth: authClient});
      let pubsub = google.pubsub({version: 'v1'});

      // Create a new pubsub topic for this enterprise.
      // [?]: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.topics/create
      await pubsub.projects.topics.create({
        name: fullPubSubTopicName,
        requestBody: {
          messageRetentionDuration: '86400s'// 24 hours
        }
      });

      // [?]: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.topics/getIamPolicy
      // Retrieve the IAM policy for the created pubsub topic.
      let getIamPolicyResponse = await pubsub.projects.topics.getIamPolicy({
        resource: fullPubSubTopicName,
      });
      let newPubSubTopicIamPolicy = getIamPolicyResponse.data;

      // Default the policy bindings to an empty array if it is not set.
      newPubSubTopicIamPolicy.bindings = newPubSubTopicIamPolicy.bindings || [];
      // Add the Fleet android MDM service account to the policy bindings.
      newPubSubTopicIamPolicy.bindings.push({
        role: 'roles/pubsub.publisher',
        members: [sails.config.custom.androidEnterpriseServiceAccountEmailAddress]
      });

      // Update the pubsub topic's IAM policy
      // [?]: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.topics/setIamPolicy
      await pubsub.projects.topics.setIamPolicy({
        resource: fullPubSubTopicName,
        requestBody: {
          policy: newPubSubTopicIamPolicy
        }
      });

      let newSubscriptionName = `projects/${sails.config.custom.androidManagementProjectId}/subscriptions/${newPubSubTopicName}`;
      // Create a new subscription for the created pubsub topic.
      // [?]: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions/create
      await pubsub.projects.subscriptions.create({
        name: newSubscriptionName,
        requestBody: {
          topic: fullPubSubTopicName,
          ackDeadlineSeconds: 60,
          messageRetentionDuration: '86400s',// 24 hours
          pushConfig: {
            pushEndpoint: pubsubPushUrl// Use the pubsubPushUrl provided by the Fleet server.
          }
        }
      });

      // Now create the new enterprise for this Fleet server.
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises.html#create
      let createEnterpriseResponse = await androidmanagement.enterprises.create({
        agreementAccepted: true,
        enterpriseToken: enterpriseToken,
        projectId: sails.config.custom.androidManagementProjectId,
        signupUrlName: signupUrlName,
        requestBody: {
          enabledNotificationTypes: [
            'ENROLLMENT',
            'STATUS_REPORT',
            'COMMAND',
            'USAGE_LOGS'
          ],
          pubsubTopic: fullPubSubTopicName,
        },
      });
      return createEnterpriseResponse.data;
    }).intercept((err)=>{
      return new Error(`When attempting to create a new Android enterprise, an error occurred. Error: ${err}`);
    });


    let newAndroidEnterpriseId = newEnterprise.id;

    // Update the database record to include details about the created enterprise.
    await AndroidEnterprise.updateOne({ id: connectionforThisInstanceExists.id }).set({
      fleetLicenseKey: fleetLicenseKey,
      androidEnterpriseId: newAndroidEnterpriseId,
      pubsubTopicName: fullPubSubTopicName,
    });



    return {
      android_enterprise_id: newAndroidEnterpriseId,// eslint-disable-line camelcase
    };

  }


};
