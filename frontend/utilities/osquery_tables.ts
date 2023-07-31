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
export const osqueryTableNames = flatMap(osqueryTables, (table) => {
  return table.hidden ? [] : table.name;
});
