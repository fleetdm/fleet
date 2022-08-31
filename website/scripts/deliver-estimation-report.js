module.exports = {


  friendlyName: 'Deliver estimation report',


  description: 'Send estimation report to Slack.',


  exits: {

    success: {
      description: 'It worked.  The estimation report was sent to Slack.'
    },

  },


  fn: async function () {

    //   ██████╗ ███████╗████████╗    ██████╗ ███████╗██████╗  ██████╗ ██████╗ ████████╗
    //  ██╔════╝ ██╔════╝╚══██╔══╝    ██╔══██╗██╔════╝██╔══██╗██╔═══██╗██╔══██╗╚══██╔══╝
    //  ██║  ███╗█████╗     ██║       ██████╔╝█████╗  ██████╔╝██║   ██║██████╔╝   ██║
    //  ██║   ██║██╔══╝     ██║       ██╔══██╗██╔══╝  ██╔═══╝ ██║   ██║██╔══██╗   ██║
    //  ╚██████╔╝███████╗   ██║       ██║  ██║███████╗██║     ╚██████╔╝██║  ██║   ██║
    //   ╚═════╝ ╚══════╝   ╚═╝       ╚═╝  ╚═╝╚══════╝╚═╝      ╚═════╝ ╚═╝  ╚═╝   ╚═╝
    //
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
    let projects = await sails.helpers.http.get(`https://api.github.com/orgs/fleetdm/projects`, {}, baseHeaders).retry();// let projects = [];// « hack if you get rate limited and want to test beta projets

    // This nasty little hack mixes in new "beta" projects that are part of Github Projects 2.0 (beta)
    // but makes them look like normal projects from the actually-documented GitHub REST API.
    // > [?] https://docs.github.com/en/enterprise-cloud@latest/issues/trying-out-the-new-projects-experience/using-the-api-to-manage-projects#finding-the-node-id-of-a-field
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // PS. In case you have to do anything with graphql ever again, try uncommenting this.
    // ```
    // console.log(
    //   require('util').inspect(
    //     await sails.helpers.http.post(`https://api.github.com/graphql`,{
    //       query:'{organization(login: "fleetdm") {projectsNext(first: 20) {nodes {id databaseId title fields(first: 20) {nodes {id name settings}} items(first: 20) {nodes{title id fieldValues(first: 8) {nodes{value projectField{name}}} content{...on Issue {repository{name} labels(first:20) {nodes{name}} assignees(first: 10) {nodes{login}}}}}} }}}}'
    //     }, baseHeaders),
    //     {depth:null}
    //   )
    // );
    // console.log();
    // console.log();
    // console.log('-0--------------');
    // console.log();
    // // return;
    // ```
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    let graphqlHairball = await sails.helpers.http.post(`https://api.github.com/graphql`,{
      query:'{organization(login: "fleetdm") {projectsNext(first: 20) {nodes {id databaseId title fields(first: 20) {nodes {id name settings}} items(first: 20) {nodes{title id fieldValues(first: 8) {nodes{value projectField{name}}} content{...on Issue {repository{name} labels(first:20) {nodes{name}} assignees(first: 10) {nodes{login}}}}}} }}}}'
    }, baseHeaders).retry();
    projects = projects.concat(
      graphqlHairball.data.organization.projectsNext.nodes.map((betaProject) => ({
        _isBetaProject: true,// « we need this because some APIs only work for one kind of project or the other
        _betaProjectColumns: JSON.parse(_.find(betaProject.fields.nodes, {name: 'Status'}).settings).options.map((betaColumn) => ({
          _isBetaColumn: true,
          _betaStatusId: betaColumn.id,
          name: betaColumn.name,
        })),
        _betaProjectCards: betaProject.items.nodes.filter((betaCard) => (
          betaCard.content && betaCard.content.labels && betaCard.content.labels.nodes &&
          betaCard.fieldValues && _.find(betaCard.fieldValues.nodes, (fieldValueNode) => fieldValueNode.projectField.name === 'Status')
        )).map((betaCard) => ({
          _isBetaCard: true,
          _betaStatusId: _.find(betaCard.fieldValues.nodes, (fieldValueNode) => fieldValueNode.projectField.name === 'Status').value,
          labels: betaCard.content.labels.nodes
        })),
        name: betaProject.title,// « it's been renamed for some reason
        node_id: betaProject.id,// eslint-disable-line camelcase
        id: betaProject.databaseId// « the good ole ID for the rest of us ("node_id" is the graphql ID)
      }))
    );// </hack>
    // console.log(require('util').inspect(projects, {depth:null}));
    // return;

    await sails.helpers.flow.simultaneouslyForEach(projects, async(project)=>{

      estimationReport[project.name] = {};

      let columns;
      if (!project._isBetaProject) {
        columns = await sails.helpers.http.get(`https://api.github.com/projects/${project.id}/columns`, {}, baseHeaders).retry();
      } else {
        columns = project._betaProjectColumns;// [?] https://docs.github.com/en/enterprise-cloud@latest/graphql/reference/objects#projectnextitem
      }

      await sails.helpers.flow.simultaneouslyForEach(columns, async(column)=>{

        estimationReport[project.name][column.name] = 0;

        let cards;
        if (!project._isBetaProject) {
          cards = await sails.helpers.http.get(`https://api.github.com/projects/columns/${column.id}/cards`, {}, baseHeaders).retry();
        } else {
          cards = project._betaProjectCards.filter((betaCard) => betaCard._betaStatusId === column._betaStatusId);
        }

        await sails.helpers.flow.simultaneouslyForEach(cards, async(card)=>{

          // Get the number of story points associated with this card.
          let numPoints = 0;

          let labels;
          if (!project._isBetaProject) {
            if (!card.content_url) {
              // ignore "notes" (FUTURE: Maybe add some kind of sniffing for a prefix like "[5]")
              labels = [];
            } else {
              let issue = await sails.helpers.http.get(card.content_url, {}, baseHeaders).retry();
              labels = issue.labels;
            }
          } else {
            labels = card.labels;
          }

          let pointLabels = labels.filter((label)=> Number(label.name) >= 1 && Number(label.name) < Infinity);
          if (pointLabels.length >= 2) { throw new Error(`Cannot have more than one story point label, but this card ${require('util').inspect(card, {depth:null})} seems to have more than one: ${_.pluck(pointLabels,'name')}`); }
          if (pointLabels.length === 0) {
            numPoints = 0;
          } else {
            numPoints = Number(pointLabels[0].name);
          }

          estimationReport[project.name][column.name] += numPoints;
        });//∞
      });//∞
    });//∞

    //  ██████╗  ██████╗ ███████╗████████╗    ████████╗ ██████╗
    //  ██╔══██╗██╔═══██╗██╔════╝╚══██╔══╝    ╚══██╔══╝██╔═══██╗
    //  ██████╔╝██║   ██║███████╗   ██║          ██║   ██║   ██║
    //  ██╔═══╝ ██║   ██║╚════██║   ██║          ██║   ██║   ██║
    //  ██║     ╚██████╔╝███████║   ██║          ██║   ╚██████╔╝
    //  ╚═╝      ╚═════╝ ╚══════╝   ╚═╝          ╚═╝    ╚═════╝
    //
    //  ███████╗██╗      █████╗  ██████╗██╗  ██╗
    //  ██╔════╝██║     ██╔══██╗██╔════╝██║ ██╔╝
    //  ███████╗██║     ███████║██║     █████╔╝
    //  ╚════██║██║     ██╔══██║██║     ██╔═██╗
    //  ███████║███████╗██║  ██║╚██████╗██║  ██╗
    //  ╚══════╝╚══════╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝
    //
    sails.log('Delivering estimation report to Slack...');
    if (!sails.config.custom.slackWebhookUrlForGithubEstimates) {
      throw new Error(
        'Estimation report not delivered: slackWebhookUrlForGithubEstimates needs to be configured in sails.config.custom. Here\'s the undelivered report: ' +
        `${require('util').inspect(estimationReport, {depth:null})}`
      );
    } else {
      await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForGithubEstimates, {
        text: `New estimation report:\n${require('util').inspect(estimationReport, {depth:null})}`
      }).retry();
    }

  }


};

