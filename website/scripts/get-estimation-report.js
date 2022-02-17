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

    // Fetch projects
    let projects = await sails.helpers.http.get(`https://api.github.com/orgs/fleetdm/projects`, {}, baseHeaders);
    projects = projects.concat(
      // This nasty little hack mixes in new "beta" projects that are part of Github Projects 2.0 (beta)
      // but makes them look like normal projects from the actually-documented GitHub REST API.
      // > [?] https://docs.github.com/en/enterprise-cloud@latest/issues/trying-out-the-new-projects-experience/using-the-api-to-manage-projects#finding-the-node-id-of-a-field
      (
        await sails.helpers.http.post(`https://api.github.com/graphql`,{
          query:'{organization(login: "fleetdm") {projectsNext(first: 20) {nodes {id databaseId title}}}}'
        }, baseHeaders)
      ).data.organization.projectsNext.nodes.map((node)=>({
        isBetaProject: true,// « we need this because some APIs only work for one kind of project or the other
        name: node.title,// « it's been renamed for some reason
        node_id: node.id,// eslint-disable-line camelcase
        id: node.databaseId// « the good ole ID for the rest of us ("node_id" is the graphql ID)
      }))
    );

    // Loop over all our projects and run our report for each one.
    await sails.helpers.flow.simultaneouslyForEach(projects, async(project)=>{

      estimationReport[project.name] = {};
      let columns;
      if (!project.isBetaProject) {
        columns = await sails.helpers.http.get(`https://api.github.com/projects/${project.id}/columns`, {}, baseHeaders);
      } else {
        // This little hack supports "beta" projects that are part of Github Projects 2.0 (beta)
        columns = [];// todo
      }

      // Loop over columns and total up points for each.
      await sails.helpers.flow.simultaneouslyForEach(columns, async(column)=>{

        estimationReport[project.name][column.name] = 0;
        let cards;
        if (!project.isBetaProject) {
          cards = await sails.helpers.http.get(`https://api.github.com/projects/columns/${column.id}/cards`, {}, baseHeaders);
        } else {
          // This little hack supports "beta" projects that are part of Github Projects 2.0 (beta)
          cards = [];// todo
        }

        // Determine points for this card.
        await sails.helpers.flow.simultaneouslyForEach(cards, async(card)=>{

          // Get the number of story points associated with this card.
          let numPoints = 0;
          if (!card.content_url) {
            // ignore "notes" (FUTURE: Maybe add some kind of sniffing for a prefix like "[5]")
          } else {
            let issue;
            if (!project.isBetaProject) {
              issue = await sails.helpers.http.get(card.content_url, {}, baseHeaders);
            } else {
              // This little hack supports "beta" projects that are part of Github Projects 2.0 (beta)
              issue = {labels: []};// todo
            }
            let pointLabels = issue.labels.filter((label)=> Number(label.name) >= 1 && Number(label.name) < Infinity);
            if (pointLabels.length >= 2) { throw new Error(`Cannot have more than one story point label, but issue #${issue.id} seems to have more than one: ${_.pluck(pointLabels,'name')}`); }
            if (pointLabels.length === 0) {
              numPoints = 0;
            } else {
              numPoints = Number(pointLabels[0].name);
            }
          }
          estimationReport[project.name][column.name] += numPoints;
        });//∞
      });//∞
    });//∞

    return estimationReport;

  }


};

