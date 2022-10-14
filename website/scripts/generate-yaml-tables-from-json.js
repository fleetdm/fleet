module.exports = {


  friendlyName: 'Generate yaml tables from json',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run generate-yaml-tables-from-json`)');
    let path = require('path');
    let YAML = require('yaml');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let fleetOverridesForTables = await sails.helpers.fs.readJson(path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json'));
    let tableYamlOutputPath = path.resolve(topLvlRepoPath+'/schema/tables');


    for(let fleetOverridesTable of fleetOverridesForTables) {
      if(!fleetOverridesTable.name){
        throw new Error(`A table in the Fleet overrides schema is missing a 'name' (${fleetOverridesTable}). To resolve, add the name of this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json')}`);
      }
      let fleetOverridesYamlString = YAML.stringify(fleetOverridesTable);

      await sails.helpers.fs.write.with({
        destination: tableYamlOutputPath+'/'+fleetOverridesTable.name+'.yml',
        string: fleetOverridesYamlString,
        force: true,
      });
    }

    for(let table of await sails.helpers.fs.ls(path.resolve(topLvlRepoPath+'/schema/tables'))) {
      let tableYaml = await sails.helpers.fs.read(table);
      let parsedYamlTable;
      try {
        parsedYamlTable = YAML.parse(tableYaml);
        let fleetOverrideSchemaForThisFile = _.find(fleetOverridesForTables, {'name': parsedYamlTable.name});
        if(fleetOverrideSchemaForThisFile){
          console.log(JSON.stringify(parsedYamlTable) === JSON.stringify(fleetOverrideSchemaForThisFile));
          if(JSON.stringify(parsedYamlTable) != JSON.stringify(fleetOverrideSchemaForThisFile)){
            console.log(JSON.stringify(parsedYamlTable));
            console.log(JSON.stringify(fleetOverrideSchemaForThisFile));
          }
        }
        if(fleetOverrideSchemaForThisFile.description){
          // console.log(parsedYamlTable.description === fleetOverrideSchemaForThisFile.description);
        }
        if(fleetOverrideSchemaForThisFile.examples){
          // console.log(parsedYamlTable.examples === fleetOverrideSchemaForThisFile.examples);
        }
        if(fleetOverrideSchemaForThisFile.columns){
          if(parsedYamlTable.columns !== fleetOverrideSchemaForThisFile.columns){
            // console.log(JSON.stringify(parsedYamlTable));
            // console.log(JSON.stringify(fleetOverrideSchemaForThisFile));
            console.log(JSON.stringify(fleetOverrideSchemaForThisFile) === JSON.stringify(parsedYamlTable))
          };
        }

      } catch(e) {
        throw new Error(`Could not parse the Fleet overrides YAMl at ${table}. To resolve, make sure the YAML is valid, then try running this script again`);
      }
      fleetOverridesForTables.push(parsedYamlTable);
    }




  }


};

