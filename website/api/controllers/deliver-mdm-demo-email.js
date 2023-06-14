module.exports = {


  friendlyName: 'Deliver MDM demo video email',


  description: 'Sends an email address containing a link to a MDM demo video to the specified email address',

  extendedDescription: 'This action is triggered by form submissions on the /device-management page'

  inputs: {
    emailAddress: {
      description: 'The email address provided when this user requested access to the MDM demo video',
      type: 'string',
      required: true,
    },
    hasOverOneThousandHosts: {
      description: 'Whether the user who requested access to the MDM demo video has over 1,000 hosts.',
      type: 'boolean',
      required: true,
    }
  },


  exits: {
    success: {
      description: 'An MDM demo video email was successfully sent'
    }
  },


  fn: async function ({emailAddress, hasOverOneThousandHosts}) {

    if(hasOverOneThousandHosts) {
      // TODO are we creating leads from this form?
    }
    // Send an email to the user, with the result from the mdm-gen-cert command attached as a plain text file.
    await sails.helpers.sendTemplateEmail.with({
      to: emailAddress,
      subject: 'Dave’s MDM video (again)',
      from: sails.config.custom.fromEmailAddress,
      fromName: sails.config.custom.fromName,
      template: 'email-mdm-video',
      templateData: {}
    }).intercept((err)=>{
      return new Error(`When trying to send a MDM demo video email for a user with the email address ${emailAddress}, an error occured. full error: ${err.stack}`);
    });

    // All done.
    return;

  }


};
