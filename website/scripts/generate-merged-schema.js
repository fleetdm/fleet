module.exports = {


  friendlyName: 'Generate merged schema',


  description: 'Merge the osquery schema from the osquery/osquery-site GitHub repo with Fleet\'s overrides (/schema/tables/) and save the merged schema to /schema/osquery_fleet_schema.json',



  fn: async function () {
    let path = require('path');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let mergedSchemaOutputPath = path.resolve(topLvlRepoPath+'/schema', 'osquery_fleet_schema.json');

    let mergedSchemaTables = await sails.helpers.getExtendedOsquerySchema();

    // Save the merged schema to /schema/merged_schema.json. Note: If this file already exists, it will be overwritten.
    await sails.helpers.fs.writeJson.with({
      destination: mergedSchemaOutputPath,
      json: mergedSchemaTables,
      force: true
    });

    sails.log(`osquery schema successfully merged with Fleet\'s overrides. The merged schema has been saved at ${mergedSchemaOutputPath}`);
  }

};

