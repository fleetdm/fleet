module.exports = {


  friendlyName: 'Get estimation report',


  description: '',


  exits: {

    success: {
      outputFriendlyName: 'Estimation report',
      outputType: {}
    },

  },


  fn: async function () {

    sails.log('Getting estimation report...');

    if (!sails.config.custom.githubAccessToken) {
      throw new Error('No GitHub access token configured!  (Please set `sails.config.custom.githubAccessToken`.)');
    }//•

    let baseHeaders = {
      'User-Agent': 'fleet story points',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    let estimationReport = {};

    let projects = await sails.helpers.http.get(`https://api.github.com/orgs/fleetdm/projects`, {}, baseHeaders);
    // FUTURE: only look at particular projects here instead of all of them
    projects = projects.filter((project)=> Number(project.id) === 13160610);// « TODO get rid of this hack

    for (let project of projects) {// TODO: do this in parallel
      estimationReport[project.name] = {};
      let columns = await sails.helpers.http.get(`https://api.github.com/projects/${project.id}/columns`, {}, baseHeaders);
      for (let column of columns) {// TODO: do this in parallel
        // console.log('------',project.name, column.name);
        estimationReport[project.name][column.name] = 0;
        let cards = await sails.helpers.http.get(`https://api.github.com/projects/columns/${column.id}/cards`, {}, baseHeaders);
        for (let card of cards) {// TODO: do this in parallel

          // Get the number of story points associated with this card.
          let numPoints = 0;
          if (!card.content_url) {
            // ignore "notes" (FUTURE: Maybe add some kind of sniffing for a prefix like "[5]")
          } else {
            // let ownerAndRepo = 'fleetdm/fleet';
            // let issueId = 4168;
            // let issue = await sails.helpers.http.get(`https://api.github.com/repos/${ownerAndRepo}/issues/${issueId}`, {}, baseHeaders);
            let issue = await sails.helpers.http.get(card.content_url, {}, baseHeaders);
            let pointLabels = issue.labels.filter((label)=> Number(label.name) >= 1 && Number(label.name) < Infinity);
            if (pointLabels.length >= 2) { throw new Error(`Cannot have more than one story point label, but issue #${issue.id} seems to have more than one: ${_.pluck(pointLabels,'name')}`); }
            if (pointLabels.length === 0) {
              numPoints = 0;
            } else {
              numPoints = Number(pointLabels[0].name);
            }
          }
          // console.log(`${column.name} :: ${card.id} ::   +${numPoints}`);
          estimationReport[project.name][column.name] += numPoints;
        }//∞
      }//∞
    }//∞


    return estimationReport;

  }


};

