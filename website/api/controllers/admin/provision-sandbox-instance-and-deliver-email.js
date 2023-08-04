module.exports = {


  friendlyName: 'Provision sandbox instance and deliver email',


  description: 'Provisions a Fleet sandbox for a user and delivers an email to a user letting them know their Fleet Sandbox instance is ready.',


  inputs: {
    userId: {
      type: 'number',
      description: 'The database ID of the user who is currently on the Fleet Sandbox waitlist',
      required: true
    }
  },


  exits: {
    success: {
      description: 'A user was successfully removed from the Fleet Sandbox waitlist.'
    },

    couldNotProvisionInstance:{
      description: 'Could not provision a Fleet Sandbox Instance for this user',
    }
  },


  fn: async function ({userId}) {

    let userToRemoveFromSandboxWaitlist = await User.findOne({id: userId});

    if(!userToRemoveFromSandboxWaitlist.inSandboxWaitlist) {
      throw new Error(`When attempting to provision a Fleet Sandbox instance for a user (id:${userId}) who is on the waitlist, the user account associated with the provided ID has already been removed from the waitlist.`);
    }

    let sandboxInstanceDetails = await sails.helpers.getNewFleetSandboxInstance.with({
      firstName: userToRemoveFromSandboxWaitlist.firstName,
      lastName: userToRemoveFromSandboxWaitlist.lastName,
      emailAddress: userToRemoveFromSandboxWaitlist.emailAddress,
    })
    .intercept((err)=>{
      return new Error(`When attempting to provision a new Fleet Sandbox instance for a User (id:${userToRemoveFromSandboxWaitlist.id}), an error occured. Full error: ${err}`);
    });

    await User.updateOne({id: userId}).set({
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

    // All done.
    return;

  }


};
