module.exports = {


  friendlyName: 'View vital details',


  description: 'Display "Vital details" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this query in the library.', example: 'get-macos-disk-free-space-percentage' },
  },


  exits: {
    success: { viewTemplatePath: 'pages/vital-details' },
    notFound: { responseType: 'notFound' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function ({ slug }) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw {badConfig: 'builtStaticContent.queries'};
    } else if (!_.isString(sails.config.builtStaticContent.queryLibraryYmlRepoPath)) {
      throw {badConfig: 'builtStaticContent.queryLibraryYmlRepoPath'};
    }

    // Serve appropriate content for vital.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let thisVital = _.find(sails.config.builtStaticContent.queries, { slug: slug });
    if (!thisVital) {
      throw 'notFound';
    }

    // Find the related osquery table documentation for tables used in this query, and grab the keywordsForSyntaxHighlighting from each table used.
    let allTablesInformation = _.filter(sails.config.builtStaticContent.markdownPages, (pageInfo)=>{
      return _.startsWith(pageInfo.url, '/tables/');
    });
    // Get all the osquery table names, we'll use this list to determine which tables are used.
    let allTableNames = _.pluck(allTablesInformation, 'title');
    // Create an array of words in the vital.
    let queryWords = _.words(thisVital.query, /[^ \n;]+/g);
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

    // Setting the meta title and description of this page using the query object, and falling back to a generic title or description if vital.name or vital.description are missing.
    let pageTitleForMeta = thisVital.name ? thisVital.name + ' | Vital details' : 'Vital details';
    let pageDescriptionForMeta = thisVital.description ? thisVital.description : 'Explore Fleet’s built-in queries for collecting and storing important device information.';
    let vitals = _.where(sails.config.builtStaticContent.queries, {kind: 'built-in'});
    let macOsVitals = _.filter(vitals, (vital)=>{
      let platformsForThisPolicy = vital.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'darwin');
    });
    let windowsVitals = _.filter(vitals, (vital)=>{
      let platformsForThisPolicy = vital.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'windows');
    });
    let linuxVitals = _.filter(vitals, (vital)=>{
      let platformsForThisPolicy = vital.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'linux');
    });
    let chromeVitals = _.filter(vitals, (vital)=>{
      let platformsForThisPolicy = vital.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'chrome');
    });
    // Respond with view.
    return {
      thisVital,
      macOsVitals,
      windowsVitals,
      linuxVitals,
      chromeVitals,
      queryLibraryYmlRepoPath: sails.config.builtStaticContent.queryLibraryYmlRepoPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
      columnNamesForSyntaxHighlighting,
      tableNamesForSyntaxHighlighting,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};
