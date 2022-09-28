module.exports = {


  friendlyName: 'Build merged schema',


  description: 'Merges the osquery schema located at /frontend/osquery_tables.json with Fleet\'s overrides (/schema/fleet_schema.json) and save the merged schema to /schema/merged_schema.json',



  fn: async function () {
    let path = require('path');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let mergedSchemaOutputPath = path.resolve(topLvlRepoPath+'/schema', 'merged_schema.json');

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

