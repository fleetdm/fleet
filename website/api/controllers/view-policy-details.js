module.exports = {


  friendlyName: 'View policy detail',


  description: 'Display "policy details" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this policy in the library.', example: 'get-macos-disk-free-space-percentage' },
  },


  exits: {
    success: { viewTemplatePath: 'pages/policy-details' },
    notFound: { responseType: 'notFound' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function ({ slug }) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.policies)) {
      throw {badConfig: 'builtStaticContent.policies'};
    } else if (!_.isString(sails.config.builtStaticContent.policyLibraryYmlRepoPath)) {
      throw {badConfig: 'builtStaticContent.queryLibraryYmlRepoPath'};
    }

    // Serve appropriate content for policy.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let policy = _.find(sails.config.builtStaticContent.policies, { slug: slug });
    if (!policy) {
      throw 'notFound';
    }

    // Find the related osquery table documentation for tables used in this query, and grab the keywordsForSyntaxHighlighting from each table used.
    let allTablesInformation = _.filter(sails.config.builtStaticContent.markdownPages, (pageInfo)=>{
      return _.startsWith(pageInfo.url, '/tables/');
    });
    // Get all the osquery table names, we'll use this list to determine which tables are used.
    let allTableNames = _.pluck(allTablesInformation, 'title');
    // Create an array of words in the query.
    let queryWords = _.words(policy.query, /[^ ]+/g);
    let columnNamesForSyntaxHighlighting = [];
    let tableNamesForSyntaxHighlighting = [];
    // Get all of the words that appear in both arrays
    let intersectionBetweenQueryWordsAndTableNames = _.intersection(queryWords, allTableNames);
    // For each matched osquery table, add the keywordsForSyntaxHighlighting and the names of the tables used into two arrays.
    for(let tableName of intersectionBetweenQueryWordsAndTableNames) {
      let tableMentionedInThisQuery = _.find(sails.config.builtStaticContent.markdownPages, {title: tableName});
      tableNamesForSyntaxHighlighting.push(tableMentionedInThisQuery.title);
      let keyWordsForThisTable = tableMentionedInThisQuery.keywordsForSyntaxHighlighting;
      columnNamesForSyntaxHighlighting = columnNamesForSyntaxHighlighting.concat(keyWordsForThisTable);
    }
    // Remove the table names from the array of column names to highlight.
    columnNamesForSyntaxHighlighting = _.difference(columnNamesForSyntaxHighlighting, tableNamesForSyntaxHighlighting);


    // Setting the meta title and description of this page using the query object, and falling back to a generic title or description if policy.name or policy.description are missing.
    let pageTitleForMeta = policy.name ? policy.name + ' | Query details' : 'Query details';
    let pageDescriptionForMeta = policy.description ? policy.description : 'View more information about a query in Fleet\'s standard query library';
    // Respond with view.
    return {
      policy,
      queryLibraryYmlRepoPath: sails.config.builtStaticContent.policyLibraryYmlRepoPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
      columnNamesForSyntaxHighlighting,
      tableNamesForSyntaxHighlighting,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};
