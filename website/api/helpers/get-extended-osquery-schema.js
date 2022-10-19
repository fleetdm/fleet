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
    let YAML = require('yaml');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let VERSION_OF_OSQUERY_SCHEMA_TO_USE = '5.4.0';
    // Getting the specified osquery schema from the osquery/osquery-site GitHub repo.
    let rawOsqueryTables = await sails.helpers.http.get('https://raw.githubusercontent.com/osquery/osquery-site/source/src/data/osquery_schema_versions/'+VERSION_OF_OSQUERY_SCHEMA_TO_USE+'.json');

    let fleetOverridesForTables = [];

    let filesInTablesFolder = await sails.helpers.fs.ls(path.resolve(topLvlRepoPath+'/schema/tables'));

    let yamlSchemaInTablesFolder = filesInTablesFolder.filter((filename)=>{return _.endsWith(filename, '.yml');});

    for(let yamlSchema of yamlSchemaInTablesFolder) {
      let tableYaml = await sails.helpers.fs.read(yamlSchema);
      let parsedYamlTable;
      try {
        parsedYamlTable = YAML.parse(tableYaml, {prettyErrors: true});
      } catch(err) {
        throw new Error(`Could not parse the Fleet overrides YAMl at ${yamlSchema} on line ${err.linePos.start.line}. To resolve, make sure the YAML is valid, then try running this script again`+err.stack);
      }
      if(parsedYamlTable.name) {
        if(typeof parsedYamlTable.name !== 'string') {
          throw new Error(`Could not merge osquery schema with Fleet overrides. A table in the Fleet overrides schema has an invalid "name" (Expected a string, but instead got a ${typeof parsedYamlTable.name}. To resolve, change the "name" of the table located at ${yamlSchema} to be a string.`);
        }
        fleetOverridesForTables.push(parsedYamlTable);
      } else { // Throw an error if a Fleet override table is missing a "name".
        throw new Error(`Could not merge osquery schema with Fleet overrides. A table in the Fleet overrides schema is missing a "name". To resolve, add a "name" to the Fleet override table located at ${yamlSchema}.`);
      }
    }

    let expandedTables = []; // create an empty array for the merged schema.

    for(let osquerySchemaTable of rawOsqueryTables) {

      let fleetOverridesForTable = _.find(fleetOverridesForTables, {'name': osquerySchemaTable.name}); // Setting a flag if this table exists in the Fleet overrrides JSON
      let expandedTableToPush = _.clone(osquerySchemaTable);

      if(!fleetOverridesForTable) {
        if(_.endsWith(osquerySchemaTable.name, '_events')) {// Make sure that all tables that have names ending in '_events' have evented: true
          expandedTableToPush.evented = true;// FUTURE: fix this in the main osquery schema so that they always have evented: true
        }
        if(expandedTableToPush.url) { // Set the osqueryRepoUrl to be the table's original url.
          expandedTableToPush.osqueryRepoUrl = expandedTableToPush.url;
        }
        // Set the URL of the table to be the table's page on fleetdm.com
        expandedTableToPush.url = 'https://fleetdm.com/tables/'+encodeURIComponent(expandedTableToPush.name);
        // Since we don't have a Fleet override for this table, we'll set the fleetRepoUrl for this table to be a link to create the Fleet override table YAML.
        // This is done by adding a 'filename' and 'value' as search parameters to a url that creates a new folder in the schema/tables/ folder.
        let sampleYamlSchemaForThisTable =`name: ${expandedTableToPush.name}\ndescription: >- # (required) string - The description for this table. Note: this field supports markdown\n\t# Add description here\nexamples: >- # (optional) string - An example query for this table. Note: This field supports markdown\n\t# Add examples here\nnotes: >- # (optional) string - Notes about this table. Note: This field supports markdown.\n\t# Add notes here\ncolumns: # (required)\n\t- name: # (required) string - The name of the column\n\t  description: # (required) string - The column's description\n\t  type: # (required) string - the column's data type\n\t  required: # (required) boolean - whether or not this column is required to query this table.`;

        expandedTableToPush.fleetRepoUrl = 'https://github.com/fleetdm/fleet/new/main/schema/tables/?filename='+encodeURIComponent(expandedTableToPush.name)+'.yml&value='+encodeURIComponent(sampleYamlSchemaForThisTable);

        expandedTables.push(expandedTableToPush);
      } else { // If this table exists in the Fleet overrides schema, we'll override the values
        if(fleetOverridesForTable.platforms !== undefined) {
          if(!_.isArray(fleetOverridesForTable.platforms)) {
            throw new Error(`Could not merge osquery schema with Fleet overrides. The Fleet override for the "${fleetOverridesForTable.name}" table located at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverridesForTable.name+'.yml')} has an invalid "platforms" value. To resolve, change the "platforms" for this table to be an array of values.`);
          } else{
            expandedTableToPush.platforms = _.clone(fleetOverridesForTable.platforms);
          }
        }
        if(fleetOverridesForTable.description !== undefined){
          if(typeof fleetOverridesForTable.description !== 'string') {
            throw new Error(`Could not merge osquery schema with Fleet overrides. The Fleet override for the "${fleetOverridesForTable.name}" table located at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverridesForTable.name+'.yml')} has an invalid "description". To resolve, change the "description" for this table to be a string.`);
          } else {
            expandedTableToPush.description = _.clone(fleetOverridesForTable.description);
          }
        }
        if(fleetOverridesForTable.examples !== undefined) {
          if(typeof fleetOverridesForTable.examples !== 'string') {
            throw new Error(`Could not merge osquery schema with Fleet overrides. The Fleet override for the "${fleetOverridesForTable.name}" table located at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverridesForTable.name+'.yml')} has an invalid "examples". To resolve, change the "examples" for this table to be a string.`);
          } else {
            expandedTableToPush.examples = _.clone(fleetOverridesForTable.examples);
          }
        }
        if(fleetOverridesForTable.notes !== undefined) {
          if(typeof fleetOverridesForTable.notes !== 'string') {
            throw new Error(`Could not merge osquery schema with Fleet overrides. The Fleet override for the "${fleetOverridesForTable.name}" table located at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverridesForTable.name+'.yml')} has an invalid "notes". To resolve, change the "notes" for this table to be a string.`);
          } else {
            expandedTableToPush.notes = _.clone(fleetOverridesForTable.notes);
          }
        }
        // If the table has Fleet overrides, we'll add the URL of the YAML file in the Fleet Github repo as the `fleetRepoUrl`, and add set the url to be where this table will live on fleetdm.com.
        expandedTableToPush.fleetRepoUrl = 'https://github.com/fleetdm/fleet/blob/main/schema/tables/'+encodeURIComponent(expandedTableToPush.name)+'.yml';
        expandedTableToPush.url = 'https://fleetdm.com/tables/'+encodeURIComponent(expandedTableToPush.name);
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
          if(!_.isArray(fleetOverridesForTable.columns)){
            throw new Error(`The osquery schema could not be merged with the Fleet overrrides. The "${fleetOverridesForTable.name}" table in Fleet's overrides has an invalid "columns". To resolve, change the "columns" to be an array of objects (each containing values for "name", "type", "description" and "required" properties), and try running the script again.`);
          }
          for(let fleetOverrideColumn of fleetOverridesForTable.columns) {
            if(!fleetOverrideColumn.name) {
              throw new Error(`The osquery schema could not be merged with the Fleet overrides. A column in the "${fleetOverridesForTable.name}" is missing a "name". To resolve, make sure every column in /schema/tables/${fleetOverridesForTable.name}.yml has a "name" property`);
            }
            let columnExistsInBothSchemas = _.find(osquerySchemaTable.columns, {'name': fleetOverrideColumn.name});
            if(!columnExistsInBothSchemas) {
              let overrideColumnToAdd = _.clone(fleetOverrideColumn);
              // Make sure the column we're adding has all the information we need, if it is missing a description or a type, we'll throw an error.

              if(overrideColumnToAdd.description) {
                if (typeof overrideColumnToAdd.description !== 'string') {
                  throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "description" for the "${fleetOverrideColumn.name}" column of the "${fleetOverridesForTable.name}" table is an invalid type (${typeof fleetOverrideColumn.name}). to resolve, change the column's "description" to be a string.`);
                }//•
              } else {
                throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "${fleetOverrideColumn.name}" column added to the merged schema for the "${fleetOverridesForTable.name}" table is missing a description in the Fleet overrides schema. To resolve, add a description for this column to the Fleet overrides schema.`);
              }

              if(overrideColumnToAdd.type) {
                if(typeof overrideColumnToAdd.type !== 'string') {
                  throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "type" for the "${fleetOverrideColumn.name}" column of the "${fleetOverridesForTable.name}" table is an invalid type (${typeof fleetOverrideColumn.type}). To resolve, change the value of a column's "type" to be a string.`);
                }//•
              } else {
                throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "${fleetOverrideColumn.name}" column added to the merged schema for the "${fleetOverridesForTable.name}" table is missing a "type" in the Fleet overrides schema. To resolve, add a type for this column to the Fleet overrides schema.`);
              }

              if(overrideColumnToAdd.platforms) {
                if(!_.isArray(overrideColumnToAdd.platforms)) {
                  throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "platforms" property of the "${overrideColumnToAdd.name}" column of the "${fleetOverridesForTable.name}" table has an invalid value. To resolve, change the "platforms" of this column to an array`);
                }//•
              }

              if(overrideColumnToAdd.required === undefined) {
                throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "${fleetOverrideColumn.name}" column added in the Fleet overrides for the "${fleetOverridesForTable.name}" table is missing a "required" value. To resolve, add a "required" value (a boolean) to the column in Fleet's overrides at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverridesForTable.name+'.yml')}`);
              } else if(typeof overrideColumnToAdd.required !== 'boolean') {
                throw new Error(`The osquery tables could not be merged with the Fleet overrides. The "${fleetOverrideColumn.name}" column added in the Fleet overrides for the "${fleetOverridesForTable.name}" table has an invalid "required" value. To resolve, change the value of the "required" property for this to the column in Fleet's overrides at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverridesForTable.name+'.yml')} to be either "true" or "false"`);
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
      let fleetOverrideToPush = _.clone(fleetOverridesForTable);
      if(!fleetOverrideToPush.name){
        throw new Error(`A table in the Fleet overrides schema is missing a 'name' (${JSON.stringify(fleetOverrideToPush)}). To resolve, make sure this table has a "name" property.`);
      }
      let fleetSchemaTableExistsInOsquerySchema = _.find(rawOsqueryTables, (table)=>{
        return fleetOverrideToPush.name === table.name;
      });
      if(!fleetSchemaTableExistsInOsquerySchema) { // If a table in the Fleet schema does not exist in the osquery schema, we'll add it to the final schema after making sure that it has the required values.

        if(!fleetOverrideToPush.description) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema, the "${fleetOverrideToPush.name}" table is missing a 'description' value. To resolve, add a description to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')}. Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at https://github.com/osquery/osquery-site/source/src/data/osquery_schema_versions/${VERSION_OF_OSQUERY_SCHEMA_TO_USE}.json`);
        }
        if(!fleetOverrideToPush.platforms) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema, the "${fleetOverrideToPush.name}" table is missing a 'platforms' value. To resolve, add an array of platforms to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')}. Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at https://github.com/osquery/osquery-site/source/src/data/osquery_schema_versions/${VERSION_OF_OSQUERY_SCHEMA_TO_USE}.json`);
        }
        if(fleetOverrideToPush.evented === undefined) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema, the "${fleetOverrideToPush.name}" table is missing a 'evented' value. To resolve, add an evented value to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')} .\n Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema https://github.com/osquery/osquery-site/source/src/data/osquery_schema_versions/${VERSION_OF_OSQUERY_SCHEMA_TO_USE}.json`);
        } else if(typeof fleetOverrideToPush.evented !== 'boolean') {
          throw new Error(`Could not add a new table from the Fleet overrides to the final merged schema. The "${fleetOverrideToPush.name}" table has an invalid "evented" value. (expected a boolean, but instead got a ${typeof fleetOverrideToPush.evented}) To resolve, change the "evented" value for this table to be true or false, then try running this script again.`);
        }
        if(!fleetOverrideToPush.columns) {
          throw new Error(`Could not add a new table from the Fleet overrides to final merged schema. The "${fleetOverrideToPush.name}" table is missing a "columns" value. To resolve, add an array of columns to this table to the Fleet overrides schema at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')}. Tip: If this table is meant to override a table in the osquery schema, you may want to check that the "name" value of the added table is the same as the table in the osquery schema located at https://github.com/osquery/osquery-site/source/src/data/osquery_schema_versions/${VERSION_OF_OSQUERY_SCHEMA_TO_USE}.json`);
        } else {

          for(let columnToValidate of fleetOverrideToPush.columns) { // Check each column in the table to make sure it has the required values, and that all values are the correct type.

            if(!columnToValidate.name) {
              throw new Error(`Could not add a new table from the Fleet overrides schema. A column in the "${fleetOverrideToPush.name}" table is missing a "name". To resolve, make sure every column in the table located at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')} has a "name" property.`);

            } else if(typeof columnToValidate.name !== 'string') {
              throw new Error(`Could not add a new table from the Fleet overrides schema. A column in the "${fleetOverrideToPush.name}" table located at /schema/tables/${fleetOverrideToPush.name}.yml has an invalid "name" (expected a string, but instead got ${typeof columnToValidate.name}).\nTo resolve, make sure that the "name" of every column in this table is a string.`);
            }//•

            if(!columnToValidate.type) {
              throw new Error(`Could not add a new table from the Fleet overrides schema. The "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table is missing a "type". To resolve add a "type" to the "${columnToValidate.name}" column at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')}.`);
            } else if(typeof columnToValidate.type !== 'string') {
              throw new Error(`Could not add a table from the Fleet overrides schema. The "type" of the "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')} has an invalid value. (expected a string, but got a ${typeof columnToValidate.type}) To resolve, change the value of the column's "type" be a string.`);
            }//•

            if(!columnToValidate.description) {
              throw new Error(`Could not add a new table from the Fleet overrides schema. The "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table is missing a "description". To resolve add a "description" property to the "${columnToValidate.name}" column at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')}`);
            } else if (typeof columnToValidate.description !== 'string') {
              throw new Error(`Could not add a table from the Fleet overrides schema. The "description" property of the "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')} has an invalid "description" value. To resolve, change the "description" property of the added column to be a string.`);
            }//•

            if(columnToValidate.required === 'undefined') {
              throw new Error(`Could not add a new table from the Fleet overrides schema. The "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table is missing a "required" property. To resolve add a "required" property to the "${columnToValidate.name}" column at ${path.resolve(topLvlRepoPath+'/schema/tables', fleetOverrideToPush.name+'.yml')}`);
            } else if (typeof columnToValidate.required !== 'boolean') {
              throw new Error(`Could not add a new table from the Fleet overrides schema. The "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table has an invalid "required" value. (expected a boolean, but instead got a ${typeof columnToValidate.required}) To resolve, change the "required" property of the added column to be a boolean.`);
            }//•

            if(columnToValidate.platforms) {
              if(!_.isArray(columnToValidate.platforms)){
                throw new Error(`Could not add a new table from the Fleet overrides schema. The "platforms" property of the "${columnToValidate.name}" column of the "${fleetOverrideToPush.name}" table has an invalid value. To resolve, change the "platforms" of this column to an array`);
              }//•
            }
          }//∞ After each column in Fleet overrides table
        }
        // After we've made sure that this table has all the required values, we'll add the url of the table's YAML file in the Fleet GitHub repo as the `fleetRepoUrl`  and the location of this table on fleetdm.com as the `url` before adding it to our merged schema.
        fleetOverrideToPush.url = 'https://fleetdm.com/tables/'+encodeURIComponent(fleetOverrideToPush.name);
        fleetOverrideToPush.fleetRepoUrl = 'https://github.com/edit/fleetdm/fleet/schema/tables/'+encodeURIComponent(fleetOverrideToPush.name)+'.yml';
        expandedTables.push(fleetOverrideToPush);
      }//∞ After each Fleet overrides table
    }
    // Return the merged schema
    return expandedTables;
  }

};

