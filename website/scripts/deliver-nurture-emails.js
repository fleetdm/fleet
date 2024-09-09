module.exports = {


  friendlyName: 'Deliver nurture emails',


  description: 'Sends nurture emails to users who have been at psychological stage 3 & 4 for more than a day, and users who have been stage five for six weeks.',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run deliver-nurture-emails`)');

    let nowAt = Date.now();
    let nurtureCampaignStartedAt = new Date('07-22-2024').getTime();
    let oneHourAgoAt = nowAt - (1000 * 60 * 60);
    let oneDayAgoAt = nowAt - (1000 * 60 * 60 * 24);
    let sixWeeksAgoAt = nowAt - (1000 * 60 * 60 * 24 * 7 * 6);
    // Find user records that are over an hour old that were created after July 22nd.
    let usersWithMdmBuyingSituation = await User.find({
      primaryBuyingSituation: 'mdm',
      createdAt: {
        '>=': nurtureCampaignStartedAt,
        '<=': oneHourAgoAt,
      },
    });

    // Only send emails to stage 3 users who have not received a nurture email for this stage, and that have been stage 3 for at least one day.
    let stageThreeMdmFocusedUsersWhoHaveNotReceivedAnEmail = _.filter(usersWithMdmBuyingSituation, (user)=>{
      return user.stageThreeNurtureEmailSentAt === 0
      && user.psychologicalStage === '3 - Intrigued';
    });

    // Only send emails to stage 4 users who have not received a a nurture email for this stage, and that have been stage 4 for at least one day.
    let stageFourMdmFocusedUsersWhoHaveNotReceivedAnEmail = _.filter(usersWithMdmBuyingSituation, (user)=>{
      return user.stageFourNurtureEmailSentAt === 0
      && user.psychologicalStage === '4 - Has use case';
    });

    // Only send emails to stage 5 users who have not received a nurture email for this stage, and that have been stage 5 for at least six weeks.
    let stageFiveMdmFocusedUsersWhoHaveNotReceivedAnEmail = _.filter(usersWithMdmBuyingSituation, (user)=>{
      return user.stageFiveNurtureEmailSentAt === 0
      && user.psychologicalStage === '5 - Personally confident';
    });

    let emailedStageThreeUserIds = [];
    for(let user of stageThreeMdmFocusedUsersWhoHaveNotReceivedAnEmail) {
      if(user.psychologicalStageLastChangedAt > oneDayAgoAt) {
        continue;
      } else {
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-nurture-stage-three',
          layout: 'layout-nurture-email',
          templateData: {
            firstName: user.firstName,
            emailAddress: user.emailAddress
          },
          to: user.emailAddress,
          toName: `${user.firstName} ${user.lastName}`,
          subject: 'Was it any good?',
          bcc: [sails.config.custom.activityCaptureEmailForNutureEmails],
          from: sails.config.custom.contactEmailForNutureEmails,
          fromName: sails.config.custom.contactNameForNurtureEmails,
          ensureAck: true,
        });
        emailedStageThreeUserIds.push(user.id);
      }
    }

    await User.update({id: {in: emailedStageThreeUserIds}})
    .set({
      stageThreeNurtureEmailSentAt: nowAt,
    });

    let emailedStageFourUserIds = [];
    for(let user of stageFourMdmFocusedUsersWhoHaveNotReceivedAnEmail) {
      if(user.psychologicalStageLastChangedAt > oneDayAgoAt) {
        continue;
      } else {
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // Note: We commented this out because it was interfering with the ability for leads to flow
        // without making reps wait.  We can turn it back on when we have a way for Drew to disable
        // nurture emails on a per-contact basis from Salesforce.
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // await sails.helpers.sendTemplateEmail.with({
        //   template: 'email-nurture-stage-four',
        //   layout: 'layout-nurture-email',
        //   templateData: {
        //     firstName: user.firstName,
        //     emailAddress: user.emailAddress
        //   },
        //   to: user.emailAddress,
        //   toName: `${user.firstName} ${user.lastName}`,
        //   subject: 'Deploy open-source MDM',
        //   bcc: [sails.config.custom.activityCaptureEmailForNutureEmails],
        //   from: sails.config.custom.contactEmailForNutureEmails,
        //   fromName: sails.config.custom.contactNameForNurtureEmails,
        //   ensureAck: true,
        // });
        emailedStageFourUserIds.push(user.id);
      }
    }

    await User.update({id: {in: emailedStageFourUserIds}})
    .set({
      stageFourNurtureEmailSentAt: nowAt,
    });


    let emailedStageFiveUserIds = [];
    for(let user of stageFiveMdmFocusedUsersWhoHaveNotReceivedAnEmail) {
      if(user.psychologicalStageLastChangedAt > sixWeeksAgoAt) {
        continue;
      } else {
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-nurture-stage-five',
          layout: 'layout-nurture-email',
          templateData: {
            firstName: user.firstName,
            emailAddress: user.emailAddress
          },
          to: user.emailAddress,
          toName: `${user.firstName} ${user.lastName}`,
          subject: 'Update',
          bcc: [sails.config.custom.activityCaptureEmailForNutureEmails],
          from: sails.config.custom.contactEmailForNutureEmails,
          fromName: sails.config.custom.contactNameForNurtureEmails,
          ensureAck: true,
        });
        emailedStageFiveUserIds.push(user.id);
      }
    }

    await User.update({id: {in: emailedStageFiveUserIds}})
    .set({
      stageFiveNurtureEmailSentAt: nowAt,
    });

  }


};

