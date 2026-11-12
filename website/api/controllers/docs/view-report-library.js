module.exports = {


  friendlyName: 'View report library',


  description: 'Display "Report library" page.',


  exits: {
    success: { viewTemplatePath: 'pages/docs/report-library' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw {badConfig: 'builtStaticContent.queries'};
    }
    let reports = _.where(sails.config.builtStaticContent.queries, {kind: 'query'});
    let macOsReports = _.filter(reports, (report)=>{
      let platformsForThisReport = report.platform.split(', ');
      return _.includes(platformsForThisReport, 'darwin');
    });
    let windowsReports = _.filter(reports, (report)=>{
      let platformsForThisReport = report.platform.split(', ');
      return _.includes(platformsForThisReport, 'windows');
    });
    let linuxReports = _.filter(reports, (report)=>{
      let platformsForThisReport = report.platform.split(', ');
      return _.includes(platformsForThisReport, 'linux');
    });
    // Respond with view.
    return {
      macOsReports,
      windowsReports,
      linuxReports,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};
