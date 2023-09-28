module.exports = {


  friendlyName: 'Provision Sandbox instance for one user and deliver email.',


  description: 'Provisions a new Fleet Sandbox instance for a user on the Fleet Sandbox waitlist, and sends an email to the user.',

  extendedDescription: 'This script will provision a Sandbox instance for the user who has been on the waitlist the longest.',


  fn: async function () {


    let earliestCreatedUserCurrentlyOnWaitlist = await User.find({inSandboxWaitlist: true})
    .limit(1)
    .sort('createdAt ASC');

    // If there are no users on the Fleet sandbox waitlist, end the script.
    if(earliestCreatedUserCurrentlyOnWaitlist.length === 0){
      sails.log('There are no users currently waiting on the Fleet Sandbox Waitlist.');
      return;
    }

    let userToRemoveFromSandboxWaitlist = earliestCreatedUserCurrentlyOnWaitlist[0];

    let sandboxInstanceDetails = await sails.helpers.fleetSandboxCloudProvisioner.provisionNewFleetSandboxInstance.with({
      firstName: userToRemoveFromSandboxWaitlist.firstName,
      lastName: userToRemoveFromSandboxWaitlist.lastName,
      emailAddress: userToRemoveFromSandboxWaitlist.emailAddress,
    })
    .intercept((err)=>{
      return new Error(`When attempting to provision a new Fleet Sandbox instance for a User (id:${userToRemoveFromSandboxWaitlist.id}), an error occured. Full error: ${err}`);
    });


    await User.updateOne({id: userToRemoveFromSandboxWaitlist.id})
    .set({
      fleetSandboxURL: sandboxInstanceDetails.fleetSandboxURL,
      fleetSandboxExpiresAt: sandboxInstanceDetails.fleetSandboxExpiresAt,
      fleetSandboxDemoKey: sandboxInstanceDetails.fleetSandboxDemoKey,
      inSandboxWaitlist: false,
    });


    // Send the user an email to let them know that their Fleet sandbox instance is ready.
    await sails.helpers.sendTemplateEmail.with({
      to: userToRemoveFromSandboxWaitlist.emailAddress,
      from: sails.config.custom.fromEmailAddress,
      fromName: sails.config.custom.fromName,
      subject: 'Your Fleet Sandbox instance is ready!',
      template: 'email-sandbox-ready-approved',
      templateData: {},
    });

    sails.log(`Successfully removed a user (id: ${userToRemoveFromSandboxWaitlist.id}) from the Fleet Sandbox waitlist.`);

  }


};

