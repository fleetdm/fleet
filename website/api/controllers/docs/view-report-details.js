module.exports = {


  friendlyName: 'View report details',


  description: 'Display "Report details" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this query in the library.', example: 'get-macos-disk-free-space-percentage' },
  },


  exits: {
    success: { viewTemplatePath: 'pages/docs/report-details' },
    notFound: { responseType: 'notFound' },
    badConfig: { responseType: 'badConfig' },
    redirectToPolicy: {
      description: 'The requesting user has been redirected to a policy page.',
      responseType: 'redirect'
    },
  },


  fn: async function ({ slug }) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw {badConfig: 'builtStaticContent.queries'};
    } else if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.policies)) {
      throw {badConfig: 'builtStaticContent.policies'};
    } else if (!_.isString(sails.config.builtStaticContent.queryLibraryYmlRepoPath)) {
      throw {badConfig: 'builtStaticContent.queryLibraryYmlRepoPath'};
    }

    // Serve appropriate content for report.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let report = _.find(sails.config.builtStaticContent.queries, {kind: 'query', slug: slug });
    if (!report) {
      // If we didn't find a report matching this slug, check to see if there is a policy with a matching slug.
      // Note: We do this because policies used to be on /queries/* pages. This way, all old URLs that policies used to live at will still bring users to the correct page.
      let policyWithThisSlug = _.find(sails.config.builtStaticContent.policies, {kind: 'policy', slug: slug});
      if(policyWithThisSlug){
        // If we foudn a matchign policy, redirect the user.
        throw {redirectToPolicy: `/policies/${policyWithThisSlug.slug}`};
      } else {
        throw 'notFound';
      }
    }

    // Find the related osquery table documentation for tables used in this query, and grab the keywordsForSyntaxHighlighting from each table used.
    let allTablesInformation = _.filter(sails.config.builtStaticContent.markdownPages, (pageInfo)=>{
      return _.startsWith(pageInfo.url, '/tables/');
    });
    // Get all the osquery table names, we'll use this list to determine which tables are used.
    let allTableNames = _.pluck(allTablesInformation, 'title');
    // Create an array of words in the query.
    let reportWords = _.words(report.query, /[^ \n;]+/g);
    let columnNamesForSyntaxHighlighting = [];
    let tableNamesForSyntaxHighlighting = [];
    // Get all of the words that appear in both arrays
    let intersectionBetweenReportWordsAndTableNames = _.intersection(reportWords, allTableNames);
    // For each matched osquery table, add the keywordsForSyntaxHighlighting and the names of the tables used into two arrays.
    for(let tableName of intersectionBetweenReportWordsAndTableNames) {
      let tableMentionedInThisReport = _.find(sails.config.builtStaticContent.markdownPages, {title: tableName});
      tableNamesForSyntaxHighlighting.push(tableMentionedInThisReport.title);
      let keyWordsForThisTable = tableMentionedInThisReport.keywordsForSyntaxHighlighting;
      columnNamesForSyntaxHighlighting = columnNamesForSyntaxHighlighting.concat(keyWordsForThisTable);
    }
    // Remove the table names from the array of column names to highlight.
    columnNamesForSyntaxHighlighting = _.difference(columnNamesForSyntaxHighlighting, tableNamesForSyntaxHighlighting);


    // Setting the meta title and description of this page using the report object, and falling back to a generic title or description if report.name or report.description are missing.
    let pageTitleForMeta = report.name ? report.name + ' | Report details' : 'Report details';
    let pageDescriptionForMeta = report.description ? report.description : 'View more information about a report in Fleet\'s standard query library';
    // Respond with view.
    return {
      report,
      reportLibraryYmlRepoPath: sails.config.builtStaticContent.queryLibraryYmlRepoPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
      columnNamesForSyntaxHighlighting,
      tableNamesForSyntaxHighlighting,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};
