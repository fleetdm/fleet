module.exports = {


  friendlyName: 'Get human interpretation from osquery sql',


  description: '',


  inputs: {

    sql: {
      type: 'string',
      required: true
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Humanesque interpretation',
      outputDescription: 'If the call to the LLM fails, then a success response is sent with an explanation about the failure (e.g. "under heavy load", etc)',
      outputExample: {
        risks: 'TODO: rachael can put the OS update eample',
        whatWillProbablyHappenDuringMaintenance: 'TODO: same as above'
      }
    },

  },


  fn: async function ({sql}) {

    let report;// TODO: call openai
    // TODO: also do the .tolerate

    return report;

  }


};
