module.exports = {

  // TODO: Change this into "create activity" instead (and create github issues for tasks instead)
  friendlyName: 'Create task',


  description: 'Create a task for our team related to a particular account in Salesforce.',


  inputs: {
    salesforceAccountId: { type: 'string', required: true, extendedDescription: 'This account will be used to determine the assignee (owner) for this new task.  (The account\'s owner will also be the owner for the new task.)' },
    dueDate: { type: 'string', example: 'YYYY-MM-DD', extendedDescription: 'If unspecified, defaults to the current date.', regex: /^[0-9][0-9][0-9][0-9]\-[0-9][0-9]\-[0-9][0-9]$/ },
  },


  exits: {

    success: {
      extendedDescription: 'Note that this deliberately has no return value.',
    },

  },


  fn: async function ({ salesforceAccountId, dueDate }) {
    sails.log(salesforceAccountId, dueDate);
    throw new Error('Not yet implemented');

    // require('assert')(sails.config.custom.salesforceSecret);

    // let jsforce = require('jsforce');
    // let conn = new jsforce.Connection({ /* */ });
    // let userInfo = await conn.login('___________', '____________');// TODO
    // let salesforceIntegrationUserId = userInfo.userId;

    // await conn.sobject('Task').create({
    //   DueDate: dueDate,
    //   ActivityDate: jsforce.Date.TODAY
    // });

  }


};

