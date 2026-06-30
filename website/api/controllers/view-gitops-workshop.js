module.exports = {


  friendlyName: 'View gitops workshop',


  description: 'Display "Gitops workshop" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/gitops-workshop'
    }

  },


  fn: async function () {

    let futureGitopsWorkshops = sails.futureGitopsWorkshops ? sails.futureGitopsWorkshops : [];
    // Respond with view.
    return {
      futureGitopsWorkshops
    };

  }


};
