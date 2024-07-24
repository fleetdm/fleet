module.exports = {


  friendlyName: 'Deliver nurture emails',


  description: '',


  fn: async function () {
    let nowAt = Date.now();
    let nurtureCampaignStartedAt= new Date('07-22-2024').getTime();
    let oneHourAgoAt = nowAt - (1000 * 60 * 60);
    let oneDayAgoAt = nowAt - (1000 * 60 * 60);
    let sixWeeksAgoAt = nowAt - (1000 * 60 * 60 * 24 * 7 * 6);
    sails.log('Running custom shell script... (`sails run deliver-nurture-emails`)');

    let users = await User.find({
      createdAt: {
        '>=': nurtureCampaignStartedAt,
        '<=': oneHourAgoAt,
      },
    });

    let usersWithMdmBuyingSituation = _.filter(users, (user)=>{
      return user.primaryBuyingSituation === 'mdm';
    });

    let stageThreeMdmFocusedUsersWhoHaveNotReceivedAnEmail = _.filter(usersWithMdmBuyingSituation, (user)=>{
      return user.stageThreeNurtureEmailSentAt === 0
      && user.psychologicalStage === '3 - Intrigued';
    });
    let stageFourMdmFocusedUsersWhoHaveNotReceivedAnEmail = _.filter(usersWithMdmBuyingSituation, (user)=>{
      return user.stageFourNurtureEmailSentAt === 0
      && user.psychologicalStage === '4 - Has use case';
    });
    let stageFiveMdmFocusedUsersWhoHaveNotReceivedAnEmail = _.filter(usersWithMdmBuyingSituation, (user)=>{
      return user.stageFiveNurtureEmailSentAt === 0
      && user.psychologicalStage === '5 - Personally confident';
    });

    for(let user of stageThreeMdmFocusedUsersWhoHaveNotReceivedAnEmail) {
      if(user.psychologicalStageLastChangedAt < oneDayAgoAt) {
        continue;
      } else {
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-nurture-stage-three',
          templateData: {
            firstName: user.firstName
          },
          to: user.emailAddress,
          toName: `${user.firstName} ${user.lastName}`,
          subject: 'Was it any good?',
          from: sails.config.custom.contactEmailForNutureEmails,
        });
      }
    }
    await User.update({id: {in: _.pluck(stageThreeMdmFocusedUsersWhoHaveNotReceivedAnEmail, 'id')}})
    .set({
      stageThreeNurtureEmailSentAt: nowAt,
    });

    for(let user of stageFourMdmFocusedUsersWhoHaveNotReceivedAnEmail) {
      if(user.psychologicalStageLastChangedAt < oneDayAgoAt) {
        continue;
      } else {
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-nurture-stage-four',
          templateData: {
            firstName: user.firstName
          },
          to: user.emailAddress,
          toName: `${user.firstName} ${user.lastName}`,
          subject: 'Deploy open-source MDM',
          from: sails.config.custom.contactEmailForNutureEmails,
        });
      }
    }
    await User.update({id: {in: _.pluck(stageFourMdmFocusedUsersWhoHaveNotReceivedAnEmail, 'id')}})
    .set({
      stageFourNurtureEmailSentAt: nowAt,
    });

    for(let user of stageFiveMdmFocusedUsersWhoHaveNotReceivedAnEmail) {
      if(user.psychologicalStageLastChangedAt < sixWeeksAgoAt) {
        continue;
      } else {
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-nurture-stage-five',
          templateData: {
            firstName: user.firstName
          },
          to: user.emailAddress,
          toName: `${user.firstName} ${user.lastName}`,
          subject: 'Update',
          from: sails.config.custom.contactEmailForNutureEmails,
        });
      }
    }
    await User.update({id: {in: _.pluck(stageFiveMdmFocusedUsersWhoHaveNotReceivedAnEmail, 'id')}})
    .set({
      stageFiveNurtureEmailSentAt: nowAt,
    });

  }


};

