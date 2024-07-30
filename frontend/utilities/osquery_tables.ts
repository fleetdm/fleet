import { flatMap } from "lodash";

import { IOsQueryTable } from "interfaces/osquery_table";

import osqueryFleetTablesJSON from "../../schema/osquery_fleet_schema.json";

// Typecasting explicity here as we are adding more rigid types such as
// OsqueryPlatform for platform names, instead of just any strings.
const queryTable = osqueryFleetTablesJSON as IOsQueryTable[];

export const osqueryTables = queryTable.sort((a, b) => {
  return a.name >= b.name ? 1 : -1;
});

// Note: Hiding tables where key hidden is set to true
export const osqueryTablesAvailable = osqueryTables.filter(
  (table) => !table.hidden
);

// Note: Hiding tables where key hidden is set to true
export const osqueryTableNames = flatMap(osqueryTables, (table) => {
  return table.hidden ? [] : table.name;
});

// Note: Hiding columns where table key hidden is set to true
export const osqueryTableColumnNames = flatMap(osqueryTables, (table) => {
  const tableColumnNames = flatMap(table.columns, (column) => column.name);
  return table.hidden ? [] : tableColumnNames;
});

// Note: Hiding columns where table key hidden is set to true
export const osqueryTableColumns = flatMap(osqueryTables, (table) => {
  return table.hidden ? [] : table.columns;
});

// Note: Hiding columns where table key hidden is set to true or if tables are defined but it doesn't include that table
export const selectedTableColumns = (selectedTables: string[]) => {
  const columnsFilteredBySelection = flatMap(osqueryTables, (table) => {
    const hideColumns = () => {
      if (table.hidden) {
        return true;
      }
      if (selectedTables.length > 0 && !selectedTables.includes(table.name)) {
        return true;
      }
      return false;
    };

    return hideColumns() ? [] : table.columns;
  });

  return columnsFilteredBySelection;
};
