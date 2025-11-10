module.exports = {


  friendlyName: 'Manage Fleet Premium trial instances',


  description: 'Provisions and sets up a configurable number of Fleet Premium instances in Render to be used by eligable users signing up for a fleetdm.com account.',


  fn: async function () {
    let util = require('util');
    sails.log('Running custom shell script... (`sails run manage-fleet-premium-trial-instances`)');

    if(!sails.config.custom.renderOwnerId){
      throw new Error(`Missing config value! Please set sails.config.custom.renderOwnerId and try running this script again.`);
    }

    if(!sails.config.custom.renderApiToken){
      throw new Error(`Missing config value! Please set sails.config.custom.renderApiToken and try running this script again.`);
    }
    if(!sails.config.custom.renderInstancePoolSize){
      throw new Error(`Missing config value! Please set sails.config.custom.renderApiToken and try running this script again.`);
    }



    let RENDER_POV_POOL_SIZE = sails.config.custom.renderInstancePoolSize;
    // Create an empty object to store caught errors. We don't want this script to stop running if there is an error with a single Vanta integration, so instead, we'll store any errors that occur and bail early for that connection if any occur, and we'll log them individually before the script is done.
    let errorReportById = {};

    // Determine how many Render POVs we need to create (if any)
    let numberOfPovRecordsReadyForAssignment = await RenderProofOfValue.count({status: 'ready for assignment', user: undefined});
    // Find any RenderProofOfValue records that have been created, but don't have any render services created for them.
    let numberOfPovRecordsWithNoRenderServices = await RenderProofOfValue.count({status: 'record created', user: undefined});
    let numberOfRenderPovToCreate = RENDER_POV_POOL_SIZE - numberOfPovRecordsReadyForAssignment - numberOfPovRecordsWithNoRenderServices;

    //  ╔═╗╦═╗╔═╗╔═╗╔╦╗╔═╗  ╔╦╗╔═╗╔╦╗╔═╗╔╗ ╔═╗╔═╗╔═╗  ╦═╗╔═╗╔═╗╔═╗╦═╗╔╦╗╔═╗
    //  ║  ╠╦╝║╣ ╠═╣ ║ ║╣    ║║╠═╣ ║ ╠═╣╠╩╗╠═╣╚═╗║╣   ╠╦╝║╣ ║  ║ ║╠╦╝ ║║╚═╗
    //  ╚═╝╩╚═╚═╝╩ ╩ ╩ ╚═╝  ═╩╝╩ ╩ ╩ ╩ ╩╚═╝╩ ╩╚═╝╚═╝  ╩╚═╚═╝╚═╝╚═╝╩╚══╩╝╚═╝
    sails.log(`${numberOfPovRecordsReadyForAssignment} ready for assignment`);
    if(numberOfPovRecordsReadyForAssignment < RENDER_POV_POOL_SIZE) {
      sails.log(`Provisioning ${numberOfRenderPovToCreate} Render instance(s)`);
      // Create an array with empty objects for each Render POV we need to create.
      // Note: We're using this approach so we can simultaneously generate the slugs for each new record that we need to create.
      let newRenderPovRecordsToCreate = Array.from({ length: numberOfRenderPovToCreate }, ()=>{return {};});
      // sails.log(`Generating slugs and creating database records for ${newRenderPovRecordsToCreate.length} new database records`);

      await sails.helpers.flow.simultaneouslyForEach(newRenderPovRecordsToCreate, async()=>{
        await sails.helpers.flow.build(async ()=>{
          let slugForThisInstance = await sails.helpers.ai.prompt.with({
            prompt: 'You are a creative developer. Return a unique, lowercase, two-word slug joined by a hyphen (e.g. "bumbling-bumblesaur"). Return only the slug as JSON string.',
            baseModel:'gpt-5-nano-2025-08-07',
            expectJson: true,
          }).retry();

          let newRecordForThisInstance = await RenderProofOfValue.create({
            slug: slugForThisInstance,
          }).fetch();
          return newRecordForThisInstance;
        }).retry('E_UNIQUE');// Retry if the generated slug is already being used by a DB record.
      });// End of simultaneouslyForEach(newRenderPovRecordsToCreate)
      // sails.log(`Records created!`);

      // Retrieve the records we just created and loop through them simutaniously.
      let renderInstancesToCreate = await RenderProofOfValue.find({status: 'record created'});
      // sails.log(renderInstancesToCreate);


      //
      //  ╔═╗╦═╗╔═╗╔═╗╔╦╗╔═╗  ╦═╗╔═╗╔╗╔╔╦╗╔═╗╦═╗  ╔═╗╔═╗╦═╗╦  ╦╦╔═╗╔═╗╔═╗
      //  ║  ╠╦╝║╣ ╠═╣ ║ ║╣   ╠╦╝║╣ ║║║ ║║║╣ ╠╦╝  ╚═╗║╣ ╠╦╝╚╗╔╝║║  ║╣ ╚═╗
      //  ╚═╝╩╚═╚═╝╩ ╩ ╩ ╚═╝  ╩╚═╚═╝╝╚╝═╩╝╚═╝╩╚═  ╚═╝╚═╝╩╚═ ╚╝ ╩╚═╝╚═╝╚═╝
      // Simultaneously create Render services for new database records.

      await sails.helpers.flow.simultaneouslyForEach(renderInstancesToCreate, async(povRecord)=>{

        let instanceIdAsString = String(povRecord.id);
        // Create a new project in render for this Fleet instance.
        let createProjectResponse = await sails.helpers.http.post.with({
          url: 'https://api.render.com/v1/projects',
          data: {
            name: povRecord.slug,
            ownerId: sails.config.custom.renderOwnerId,
            environments: [{
              name: 'Production',
              protectionStatus: 'unprotected',
            }]
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        }).tolerate((err)=>{
          errorReportById[instanceIdAsString] = new Error(`Could not create a new project for this Render POV. Error from Render API: ${util.inspect(err)}`);
        });

        sails.log(`Project ${povRecord.slug} created!`);
        // Example response:
        // {
        //   "id": "string",
        //   "createdAt": "2025-10-16T21:51:23.025Z",
        //   "updatedAt": "2025-10-16T21:51:23.025Z",
        //   "name": "string",
        //   "owner": {
        //     "id": "string",
        //     "name": "string",
        //     "email": "string",
        //     "twoFactorAuthEnabled": true,
        //     "type": "user"
        //   },
        //   "environmentIds": [
        //     "string"
        //   ]
        // }

        // If there was an error with the previous request, bail early for this instance.
        if(errorReportById[instanceIdAsString]){
          return;
        }

        let renderProjectId = createProjectResponse.id;

        // Update the database record for this POV.
        await RenderProofOfValue.updateOne({id: povRecord.id}).set({
          renderProjectId,
          status: 'provisioning'
        });
        // Get the ID of the production environment created on the new project, we'll need this later to move the created services to the project.
        let environmentId = createProjectResponse.environmentIds[0];

        // Create the Redis service for this instance:
        let createRedisResponse = await sails.helpers.http.post.with({
          url: 'https://api.render.com/v1/redis',
          data: {
            name: povRecord.slug+'-fleet-redis',
            ownerId: sails.config.custom.renderOwnerId,
            plan: 'starter',
            ipAllowList: [],
            maxmemoryPolicy: 'allkeys-lru',
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        }).tolerate((err)=>{
          errorReportById[instanceIdAsString] = new Error(`Could not create a Redis service for a new Render POV. Error from Render API: ${util.inspect(err)}`);
        });

        sails.log(`(id: ${povRecord.id}) Redis service created!`);

        if(errorReportById[instanceIdAsString]){
          return;
        }
        // example response:
        // [
        //   {
        //     "redis": {
        //       "id": "string",
        //       "createdAt": "2025-10-16T21:51:23.025Z",
        //       "updatedAt": "2025-10-16T21:51:23.025Z",
        //       "status": "creating",
        //       "region": "oregon",
        //       "plan": "free",
        //       "name": "string",
        //       "owner": {
        //         "id": "string",
        //         "name": "string",
        //         "email": "string",
        //         "twoFactorAuthEnabled": true,
        //         "type": "user"
        //       },
        //       "options": {
        //         "maxmemoryPolicy": "string"
        //       },
        //       "ipAllowList": [
        //         {
        //           "cidrBlock": "string",
        //           "description": "string"
        //         }
        //       ],
        //       "environmentId": "string",
        //       "version": "string",
        //       "dashboardUrl": "string"
        //     },
        //     "cursor": "string"
        //   }
        // ]
        let renderRedisServiceId = createRedisResponse.id;

        await RenderProofOfValue.updateOne({id: povRecord.id}).set({
          renderRedisServiceId,
        });

        let generatedMySQLPassword = await sails.helpers.strings.uuid();
        let generatedMySQLRootPassword = await sails.helpers.strings.uuid();
        // Create the MySQL service for this instance:
        let createMySQLResponse = await sails.helpers.http.post.with({
          // url: 'https://api.render.com/v1/servicess',// Intentionally causing an error to test error handling in this script.
          url: 'https://api.render.com/v1/services',
          data: {
            ownerId: sails.config.custom.renderOwnerId,
            type: 'private_service',
            name: povRecord.slug+'-fleet-mysql',
            repo: 'https://github.com/render-examples/mysql',
            autoDeploy: 'yes',
            serviceDetails: {
              plan: 'standard',
              runtime: 'docker',
              disk: {
                sizeGB: 1,
                name: 'mysql',
                mountPath: '/var/lib/mysql',
              },
            },
            envVars:[
              { key: 'MYSQL_DATABASE', value: 'fleet' },
              { key: 'MYSQL_USER', value: 'fleet' },
              { key: 'MYSQL_PASSWORD', value: generatedMySQLPassword },
              { key: 'MYSQL_ROOT_PASSWORD', value: generatedMySQLRootPassword }
            ]
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        }).tolerate((err)=>{
          errorReportById[instanceIdAsString] = new Error(`Could not create a MySQL service for a new Render POV. Error from Render API: ${util.inspect(err)}`);
        });

        if(errorReportById[instanceIdAsString]){
          return;
        }
        sails.log(`id: ${povRecord.id}) MySQL service created!`);
        // Example response (will be the same for the next API request):
        // {
        //   "service": {
        //     "id": "string",
        //     "autoDeploy": "yes",
        //     "branch": "string",
        //     "buildFilter": {
        //       "paths": [
        //         "string"
        //       ],
        //       "ignoredPaths": [
        //         "string"
        //       ]
        //     },
        //     "createdAt": "2025-10-16T21:51:23.025Z",
        //     "dashboardUrl": "string",
        //     "environmentId": "string",
        //     "imagePath": "string",
        //     "name": "string",
        //     "notifyOnFail": "default",
        //     "ownerId": "string",
        //     "registryCredential": {
        //       "id": "string",
        //       "name": "string"
        //     },
        //     "repo": "https://github.com/render-examples/flask-hello-world",
        //     "rootDir": "string",
        //     "slug": "string",
        //     "suspended": "suspended",
        //     "suspenders": [
        //       "admin"
        //     ],
        //     "type": "static_site",
        //     "updatedAt": "2025-10-16T21:51:23.025Z",
        //     "serviceDetails": {
        //       "buildCommand": "string",
        //       "parentServer": {
        //         "id": "string",
        //         "name": "string"
        //       },
        //       "publishPath": "string",
        //       "previews": {
        //         "generation": "off"
        //       },
        //       "url": "string",
        //       "buildPlan": "starter",
        //       "renderSubdomainPolicy": "enabled"
        //     }
        //   },
        //   "deployId": "string"
        // }

        let renderMySqlServiceId = createMySQLResponse.service.id;
        let renderMySqlDeployId = createMySQLResponse.deployId;

        // Update the database record for this POV.
        await RenderProofOfValue.updateOne({id: povRecord.id}).set({
          renderMySqlServiceId,
        });

        // Now provision a new *.try.fleetdm.com DNS record for this instance.
        let urlForThisInstance = `${povRecord.slug}.try.fleetdm.com`;

        await sails.helpers.http.post.with({
          url: 'https://api.github.com/repos/fleetdm/confidential/dispatches',
          data: {
            event_type: 'try-fleet-webhook',
            client_payload: {
              action: 'apply',
              workspace: povRecord.slug,
            }
          }
        }).tolerate((err)=>{
          errorReportById[instanceIdAsString] = new Error(`Could not send request to create a *.try.fleetdm.com DNS record for a new Render trial instance. Error from GitHub API: ${util.inspect(err)}`);
        });

        if(errorReportById[instanceIdAsString]){
          return;
        }



        // Note: we can create the Fleet service now, but if it is live before the mysql service is, then it won't be able to deploy.
        await sails.helpers.flow.until(async()=>{

          let getDeployResponse = await sails.helpers.http.get.with({
            url: `https://api.render.com/v1/services/${renderMySqlServiceId}/deploys/${renderMySqlDeployId}`,
            headers: {
              authorization: `Bearer ${sails.config.custom.renderApiToken}`
            },
          });
          if(getDeployResponse.status === 'live'){
            sails.log(`MySQL service is live. Now creating Fleet service....`);
            return true;
          } else {
            // sails.log(`MySQL service is not deployed yet, waiting 10 seconds before trying again....`);
            await sails.helpers.flow.pause(10000);
          }
        }, 600000);

        let ninetyDaysFromNowAt = Date.now() + (1000 * 60 * 60 * 24 * 90);

        let licenseKey = await sails.helpers.createLicenseKey.with({
          numberOfHosts: 10,
          organization: 'Render-trial-'+povRecord.slug,
          expiresAt: ninetyDaysFromNowAt,
        });


        let fleetEnvVars = [
          { key: 'FLEET_SOFTWARE_INSTALLER_STORE_DIR', value: '/opt/fleet/installers' },
          { key: 'FLEET_SERVER_PRIVATE_KEY', 'generateValue': true },
          { key: 'FLEET_SERVER_TLS', value: 'false' },
          { key: 'FLEET_LICENSE_KEY', value: licenseKey },
          { key: 'FLEET_REDIS_ADDRESS', value: `redis://${renderRedisServiceId}:6379`},
          { key: 'FLEET_MYSQL_ADDRESS', value: povRecord.slug+'-fleet-mysql:3306' },
          { key: 'FLEET_MYSQL_DATABASE', value: 'fleet' },
          { key: 'FLEET_MYSQL_USERNAME', value: 'fleet' },
          { key: 'FLEET_MYSQL_PASSWORD', value: generatedMySQLPassword },
          { key: 'PORT', value: '8080' },
          { key: 'FLEET_SERVER_ADDRESS', value: urlForThisInstance },
          { key: 'FLEET_SES_ACCESS_KEY_ID', value: sails.config.custom.renderInstanceSesSecretId},
          { key: 'FLEET_SES_SECRET_ACCESS_KEY', value: sails.config.custom.renderInstanceSesSecretKey },
          { key: 'FLEET_EMAIL_BACKEND', value: 'ses'},
          { key: 'FLEET_SES_REGION', value: 'us-east-2'},
          { key: 'FLEET_SES_SOURCE_ARN', value: `arn:aws:ses:us-east-2:564445215450:identity/${povRecord.slug}.try.fleetdm.com`},
        ];


        // Create the Fleet service for this instance:
        let createFleetResponse = await sails.helpers.http.post.with({
          url: 'https://api.render.com/v1/services',
          data: {
            ownerId: sails.config.custom.renderOwnerId,
            type: 'web_service',
            name: povRecord.slug+'-fleet',
            image: {
              ownerId: sails.config.custom.renderOwnerId,
              imagePath: 'fleetdm/fleet:latest'
            },
            autoDeploy: 'no',
            envVars: fleetEnvVars,
            serviceDetails: {
              runtime: 'image',
              healthCheckPath: '/healthz',
              plan: 'standard',
              preDeployCommand: 'fleet prepare --no-prompt=true db',
              previews: {
                generation: 'off',
              },
              disk: {
                sizeGB: 1,
                name: 'installers',
                mountPath: '/opt/fleet/installers',
              }
            }
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        }).tolerate((err)=>{
          errorReportById[instanceIdAsString] = new Error(`Could not create a Redis service for a new Render POV. Error from Render API: ${util.inspect(err)}`);
        });

        if(errorReportById[instanceIdAsString]){
          return;
        }

        sails.log(`id: ${povRecord.id}) Fleet service created!`);

        let renderFleetServiceId = createFleetResponse.service.id;

        await RenderProofOfValue.updateOne({id: povRecord.id}).set({
          renderFleetServiceId,
        });

        // Move the creaed services to the project we created for this pov.
        await sails.helpers.http.post.with({
          url: `https://api.render.com/v1/environments/${environmentId}/resources`,
          data: {
            resourceIds: [
              renderRedisServiceId,
              renderMySqlServiceId,
              renderFleetServiceId,
            ]
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        }).tolerate((err)=>{
          errorReportById[instanceIdAsString] = new Error(`Could not move services to the project created for a new Render POV. Error from Render API: ${util.inspect(err)}`);
        });

        await RenderProofOfValue.updateOne({id: povRecord.id}).set({
          instanceUrl: createFleetResponse.service.serviceDetails.url,
          status: 'ready for assignment'
        });

      });// End of simultaneouslyForEach(renderInstancesToCreate)

    }//ﬁ


    //
    //  ╔═╗╦  ╔═╗╔═╗╔╗╔╦ ╦╔═╗  ╔═╗═╗ ╦╔═╗╦╦═╗╔═╗╔╦╗  ╦╔╗╔╔═╗╔╦╗╔═╗╔╗╔╔═╗╔═╗╔═╗
    //  ║  ║  ║╣ ╠═╣║║║║ ║╠═╝  ║╣ ╔╩╦╝╠═╝║╠╦╝║╣  ║║  ║║║║╚═╗ ║ ╠═╣║║║║  ║╣ ╚═╗
    //  ╚═╝╩═╝╚═╝╩ ╩╝╚╝╚═╝╩    ╚═╝╩ ╚═╩  ╩╩╚═╚═╝═╩╝  ╩╝╚╝╚═╝ ╩ ╩ ╩╝╚╝╚═╝╚═╝╚═╝
    // Check for any instances that should be torn down during this run.
    let nowAt = Date.now();
    let expiringInstances = await RenderProofOfValue.find({status: 'in-use', renderTrialEndsAt: {'<': nowAt}}).populate('user');
    for(let expiringInstance of expiringInstances) {
      // Delete the services and the project for this expired POV.


      // Delete the MySQL service that was created for this record.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'DELETE',
        url: `https://api.render.com/v1/services/${expiringInstance.renderMySqlServiceId}`,
        headers: {
          authorization: `Bearer ${sails.config.custom.renderApiToken}`
        },
      }).tolerate((err)=>{
        sails.log.warn(`p1: When deleting a MySQL service (id: ${expiringInstance.renderMySqlServiceId}) for a Render POV that expired, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
        return;
      });


      // Delete the Redis service that was created for this record.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'DELETE',
        url: `https://api.render.com/v1/redis/${expiringInstance.renderRedisServiceId}`,
        headers: {
          authorization: `Bearer ${sails.config.custom.renderApiToken}`
        },
      }).tolerate((err)=>{
        sails.log.warn(`p1: When deleting a Redis service (id: ${expiringInstance.renderRedisServiceId}) for a Render POV that expired, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
        return;
      });


      // Delete the Fleet service that was created for this record.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'DELETE',
        url: `https://api.render.com/v1/services/${expiringInstance.renderFleetServiceId}`,
        headers: {
          authorization: `Bearer ${sails.config.custom.renderApiToken}`
        },
      }).tolerate((err)=>{
        sails.log.warn(`p1: When deleting a Fleet service (id: ${expiringInstance.renderFleetServiceId}) for a Render POV that expired, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
        return;
      });

      // Delete the Render project that was created for this record.
      await sails.helpers.http.sendHttpRequest.with({
        method: 'DELETE',
        url: `https://api.render.com/v1/projects/${expiringInstance.renderProjectId}`,
        headers: {
          authorization: `Bearer ${sails.config.custom.renderApiToken}`
        },
      }).tolerate((err)=>{
        sails.log.warn(`p1: When deleting a Render project (id: ${expiringInstance.renderProjectId}) for a Render POV that expired, the Render API returned an error. This project will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
        return;
      });

      let user = expiringInstance.user;
      // Send the user an email letting them know, and update the database record for this Render POV
      await sails.helpers.sendTemplateEmail.with({
        to: user.emailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Your Fleet trial has ended',
        template: 'email-fleet-premium-pov-trial-ended',
        layout: 'layout-nurture-email',
        templateData: {
          firstName: user.firstName,
        },
        ensureAck: true,
      });

      await RenderProofOfValue.updateOne({id: expiringInstance.id}).set({status: 'expired'});
    }//∞


    //
    //  ╦  ╔═╗╔═╗  ╔═╗╦═╗╦═╗╔═╗╦═╗╔═╗  ╔═╗╔╗╔╔╦╗  ╔═╗╦  ╔═╗╔═╗╔╗╔  ╦ ╦╔═╗
    //  ║  ║ ║║ ╦  ║╣ ╠╦╝╠╦╝║ ║╠╦╝╚═╗  ╠═╣║║║ ║║  ║  ║  ║╣ ╠═╣║║║  ║ ║╠═╝
    //  ╩═╝╚═╝╚═╝  ╚═╝╩╚═╩╚═╚═╝╩╚═╚═╝  ╩ ╩╝╚╝═╩╝  ╚═╝╩═╝╚═╝╩ ╩╝╚╝  ╚═╝╩
    for (let instanceIdAsString of Object.keys(errorReportById)) {
      if (false === errorReportById[instanceIdAsString]) {
        // If no error occured wehn setting up this POV, do nothing.
      } else {
        // If an error was logged while provisioning a Render POV, log the error as a warning.
        sails.log.warn(`When provisioning a new Render POV, an error occured. This script will clean up any services it created for this POV. Full error: ${errorReportById[instanceIdAsString]}`);

        // Clean up the services associated with this record and delete it.
        let povRecord = await RenderProofOfValue.findOne({id: instanceIdAsString});

        if(povRecord.renderMySqlServiceId) {
          // Delete the MySQL service that was created for this record.
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            url: `https://api.render.com/v1/services/${povRecord.renderMySqlServiceId}`,
            headers: {
              authorization: `Bearer ${sails.config.custom.renderApiToken}`
            },
          }).tolerate((err)=>{
            sails.log.warn(`p1: When deleting a MySQL service (id: ${povRecord.renderMySqlServiceId}) for a Render POV that encountered an error during setup, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
            return;
          });
          //
          await sails.helpers.http.post.with({
            url: 'https://api.github.com/repos/fleetdm/confidential/dispatches',
            data: {
              event_type: 'try-fleet-webhook',
              client_payload: {
                action: 'apply',
                workspace: povRecord.slug,
              }
            }
          }).tolerate((err)=>{
            errorReportById[instanceIdAsString] = new Error(`Could not send request to destroy a *.try.fleetdm.com DNS record for a Render trial instance that encountered an error during setup. Error from GitHub API: ${util.inspect(err)}`);
            return;
          });
        }
        if(povRecord.renderRedisServiceId) {
          // Delete the Redis service that was created for this record.
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            url: `https://api.render.com/v1/redis/${povRecord.renderRedisServiceId}`,
            headers: {
              authorization: `Bearer ${sails.config.custom.renderApiToken}`
            },
          }).tolerate((err)=>{
            sails.log.warn(`p1: When deleting a Redis service (id: ${povRecord.renderRedisServiceId}) for a Render POV that encountered an error during setup, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
            return;
          });
        }
        if(povRecord.renderFleetServiceId) {
          // Delete the Fleet service that was created for this record.
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            url: `https://api.render.com/v1/services/${povRecord.renderFleetServiceId}`,
            headers: {
              authorization: `Bearer ${sails.config.custom.renderApiToken}`
            },
          }).tolerate((err)=>{
            sails.log.warn(`p1: When deleting a Fleet service (id: ${povRecord.renderFleetServiceId}) for a Render POV that encountered an error during setup, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
            return;
          });
        }
        if(povRecord.renderProjectId) {
          // Delete the Render project that was created for this record.
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            url: `https://api.render.com/v1/projects/${povRecord.renderProjectId}`,
            headers: {
              authorization: `Bearer ${sails.config.custom.renderApiToken}`
            },
          }).tolerate((err)=>{
            sails.log.warn(`p1: When deleting a Render project (id: ${povRecord.renderProjectId}) for a Render POV that encountered an error during setup, the Render API returned an error. This service will need to be manually deleted in the Render dashboard. Error from Render API: ${util.inspect(err)}`);
            return;
          });
        }

        // Clean up the database record.
        await RenderProofOfValue.destroy({id: instanceIdAsString});


      }
    }//∞



  }


};

