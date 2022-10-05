module.exports = {


  friendlyName: 'Get extended osquery schema',


  description: 'Get the extended osquery schema and documentation supported by Fleet by reading the raw osquery tables and Fleet\'s overrides from disk, then returning the extended set of tables.',


  exits: {

    success: {
      outputFriendlyName: 'Extended tables',
      outputType: [ {} ],
    }


  },


  fn: async function () {
    let path = require('path');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let VERSION_OF_OSQUERY_SCHEMA_TO_USE = '5.4.0';
    // Getting the specified osquery schema from the osquery/osquery-site GitHub repo.
    let rawOsqueryTables = await sails.helpers.http.get('https://raw.githubusercontent.com/osquery/osquery-site/source/src/data/osquery_schema_versions/'+VERSION_OF_OSQUERY_SCHEMA_TO_USE+'.json');
    let fleetOverridesForTables = await sails.helpers.fs.readJson(path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json'));

    let expandedTables = []; // create an empty array for the merged schema.

    for(let osquerySchemaTable of rawOsqueryTables) {

      let fleetOverridesForTable = _.find(fleetOverridesForTables, {'name': osquerySchemaTable.name}); // Setting a flag if this table exists in the Fleet overrrides JSON
      let expandedTableToPush = _.clone(osquerySchemaTable);

      if(!fleetOverridesForTable) {
        if(_.endsWith(osquerySchemaTable.name, '_events')) {// Make sure that all tables that have names ending in '_events' have evented: true
          expandedTableToPush.evented = true;// FUTURE: fix this in the main osquery schema so that they always have evented: true
        }
        expandedTables.push(expandedTableToPush);
      } else { // If this table exists in the Fleet overrides schema, we'll override the values
        if(fleetOverridesForTable.platforms !== undefined) {
          expandedTableToPush.platforms = _.clone(fleetOverridesForTable.platforms);
        }
        if(fleetOverridesForTable.description !== undefined){
          expandedTableToPush.description = _.clone(fleetOverridesForTable.description);
        }
        if(fleetOverridesForTable.examples !== undefined) {
          expandedTableToPush.examples = _.clone(fleetOverridesForTable.examples);
        }
        if(fleetOverridesForTable.notes !== undefined) {
          expandedTableToPush.notes = _.clone(fleetOverridesForTable.notes);
        }
        let mergedTableColumns = [];
        for (let osquerySchemaColumn of osquerySchemaTable.columns) { // iterate through the columns in the osquery schema table
          if(!fleetOverridesForTable.columns) { // If there are no column overrides for this table, we'll add the column unchanged.
            mergedTableColumns.push(osquerySchemaColumn);
          } else {// If the Fleet overrides JSON has column data for this table, we'll find the matching column and use the values from the Fleet overrides in the final schema.
            let columnHasFleetOverrides = _.find(fleetOverridesForTable.columns, {'name': osquerySchemaColumn.name});
            if(!columnHasFleetOverrides) {// If this column has no Fleet overrides, we'll add it to the final schema unchanged
              let columnWithNoOverrides = _.clone(osquerySchemaColumn);
              if(osquerySchemaColumn.type !== undefined) {
                columnWithNoOverrides.type = osquerySchemaColumn.type.toUpperCase();
              }
              mergedTableColumns.push(osquerySchemaColumn);
            } else { // If this table has Fleet overrides, we'll adjust the value in the merged schema
              let fleetColumn = _.clone(osquerySchemaColumn);
              if(columnHasFleetOverrides.platforms !== undefined) {
                let platformWithNormalizedNames = [];
                for(let platform of columnHasFleetOverrides.platforms) {
                  if(platform === 'darwin') {
                    platformWithNormalizedNames.push('macOS');
                  } else {
                    platformWithNormalizedNames.push(_.capitalize(platform));
                  }
                }
                fleetColumn.platforms = platformWithNormalizedNames;
              }
              if(columnHasFleetOverrides.description !== undefined) {
                if(typeof columnHasFleetOverrides.description === 'string') {
                  fleetColumn.description = _.clone(columnHasFleetOverrides.description);
                } else {
                  fleetColumn.description = '';
                }
              }
              if(columnHasFleetOverrides.type !== undefined) {
                fleetColumn.type = _.clone(columnHasFleetOverrides.type.toUpperCase());
              }
              if(columnHasFleetOverrides.required !== undefined) {
                fleetColumn.required = _.clone(columnHasFleetOverrides.required);
              }
              if(columnHasFleetOverrides.hidden !== true) { // If the overrides don't explicitly hide a column, we'll set the value to false to make sure the column is visible on fleetdm.com
                fleetColumn.hidden = false;
              }
              mergedTableColumns.push(fleetColumn);
            }
          }
        }//∞ After each column in osquery schema table

        // Now iterate through the columns in the Fleet overrides, adding any columns that doesnt exist in the base osquery schema.
        if(fleetOverridesForTable.columns) {
          for(let fleetOverrideColumn of fleetOverridesForTable.columns) {
            let columnExistsInBothSchemas = _.find(osquerySchemaTable.columns, {'name': fleetOverrideColumn.name});
            if(!columnExistsInBothSchemas) {
              let overrideColumnToAdd = _.clone(fleetOverrideColumn);
              // Make sure the column we're adding has all the information we need, if it is missing a description or a type, we'll throw an error.
              if(!overrideColumnToAdd.description) {
                throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "${fleetOverrideColumn.name}" column added to the merged schema for the "${fleetOverridesForTable.name}" table is missing a description in the Fleet overrides schema. To resolve, add a description for this column to the Fleet overrides schema.`);
              }
              if(overrideColumnToAdd.type) {
                throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "${fleetOverrideColumn.name}" column added to the merged schema for the "${fleetOverridesForTable.name}" table is missing a "type" in the Fleet overrides schema. To resolve, add a type for this column to the Fleet overrides schema.`);
              }
              mergedTableColumns.push(overrideColumnToAdd);
            }
          }//∞ After each column in Fleet overrides table
        }
        expandedTableToPush.columns = mergedTableColumns;
        expandedTables.push(expandedTableToPush);
      }
    }//∞ After each table in osquery schema

    // After we've gone through the tables in the Osquery schema, we'll go through the tables in the Fleet schema JSON, and add any tables that don't exist in the osquery schema.
    for (let fleetOverridesForTable of fleetOverridesForTables) {
      if(!fleetOverridesForTable.name){
        throw new Error(`A table in the Fleet overrides schema is missing a 'name' (${fleetOverridesForTable}). To resolve, add the name of this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json')}`);
      }
      let fleetSchemaTableExistsInOsquerySchema = _.find(rawOsqueryTables, (table)=>{
        return fleetOverridesForTable.name === table.name;
      });
      if(!fleetSchemaTableExistsInOsquerySchema) { // If a table in the Fleet schema does not exist in the osquery schema, we'll add it to the final schema after making sure that it has the required values.
        if(!fleetOverridesForTable.description) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema, the "${fleetOverridesForTable.name}" table is missing a 'description' value. To resolve, add a description to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json')}.\n Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at ${path.resolve(topLvlRepoPath+'/frontend', 'osquery_tables.json')}.`);
        }
        if(!fleetOverridesForTable.platforms) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema, the "${fleetOverridesForTable.name}" table is missing a 'platforms' value. To resolve, add an array of platforms to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json')}.\n Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at ${path.resolve(topLvlRepoPath+'/frontend', 'osquery_tables.json')}.`);
        }
        if(!fleetOverridesForTable.evented) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema, the "${fleetOverridesForTable.name}" table is missing a 'evented' value. To resolve, add an evented value to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json')}.\n Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at ${path.resolve(topLvlRepoPath+'/frontend', 'osquery_tables.json')}.`);
        }
        if(!fleetOverridesForTable.columns) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema. The "${fleetOverridesForTable.name}" table is missing a "columns" value. To resolve, add an array of columns to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema', 'fleet_schema.json')}.\n Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at ${path.resolve(topLvlRepoPath+'/frontend', 'osquery_tables.json')}.`);
        }
        expandedTables.push(fleetOverridesForTable);
      }
    }
    // Return the merged schema
    return expandedTables;
  }

};

