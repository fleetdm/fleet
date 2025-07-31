module.exports = {


  friendlyName: 'Migrate old primary buying situation values',


  description: 'Updates the primaryBuyingSituation value of user records that have older values set.',

  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run. (No database changes will be made.)' },
  },

  fn: async function ({dry}) {
    sails.log('Running custom shell script... (`sails run migrate-old-primary-buying-situation-values`)');

    let numberOfUsersWhoWouldHaveBeenUpdated = 0;
    sails.log('Migrating users with primaryBuyingSituation === mdm.');
    await User.stream({primaryBuyingSituation: 'mdm'})
    .eachRecord(async (thisUser)=>{
      // Default to 'it-major-mdm' for users with an mdm primaryBuyingSituation.
      let newPrimaryBuyingSituationForThisUser = 'it-major-mdm';
      // If this user has filled out the get started questionnaire, check their answers to determine if they were interested in Linux MDM.
      if(thisUser.getStartedQuestionnaireAnswers !== {}){
        if(thisUser.getStartedQuestionnaireAnswers['what-do-you-manage-mdm']) {
          // If they previosuly selected linux as a response to the "What do you manage?" question, set their new primaryBuyingSituation to be 'it-gap-filler-mdm'
          if(thisUser.getStartedQuestionnaireAnswers['what-do-you-manage-mdm'].mdmUseCase === 'linux') {
            newPrimaryBuyingSituationForThisUser = 'it-gap-filler-mdm';
          }
        }
      }
      if(dry) {
        numberOfUsersWhoWouldHaveBeenUpdated++;
      } else {
        // Update this user's primary buying situation.
        await User.updateOne({id: thisUser.id}).set({
          primaryBuyingSituation: newPrimaryBuyingSituationForThisUser,
        });
      }
    });

    sails.log('Migrating users with primaryBuyingSituation === eo-it.');
    await User.stream({primaryBuyingSituation: 'eo-it'})
    .eachRecord(async (thisUser)=>{
      let newPrimaryBuyingSituationForThisUser = 'it-misc';
      if(dry) {
        numberOfUsersWhoWouldHaveBeenUpdated++;
      } else {
        // Update this user's primary buying situation.
        await User.updateOne({id: thisUser.id}).set({
          primaryBuyingSituation: newPrimaryBuyingSituationForThisUser,
        });
      }
    });

    sails.log('Migrating users with primaryBuyingSituation === eo-security.');
    await User.stream({primaryBuyingSituation: 'eo-security'})
    .eachRecord(async (thisUser)=>{
      let newPrimaryBuyingSituationForThisUser = 'security-misc';
      if(dry) {
        numberOfUsersWhoWouldHaveBeenUpdated++;
      } else {
        // Update this user's primary buying situation.
        await User.updateOne({id: thisUser.id}).set({
          primaryBuyingSituation: newPrimaryBuyingSituationForThisUser,
        });
      }
    });

    sails.log('Migrating users with primaryBuyingSituation === vm.');
    await User.stream({primaryBuyingSituation: 'vm'})
    .eachRecord(async (thisUser)=>{
      let newPrimaryBuyingSituationForThisUser = 'security-vm';
      if(dry) {
        numberOfUsersWhoWouldHaveBeenUpdated++;
      } else {
        // Update this user's primary buying situation.
        await User.updateOne({id: thisUser.id}).set({
          primaryBuyingSituation: newPrimaryBuyingSituationForThisUser,
        });
      }
    });

    if(dry){
      sails.log(`Dry run: would have migrated the primaryBuyingSituation values for ${numberOfUsersWhoWouldHaveBeenUpdated} user records.`);
    }
  }


};

