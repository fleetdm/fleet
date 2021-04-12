import { flatMap, sortBy } from "lodash";
import osqueryTablesJSON from "../osquery_tables.json";

export const normalizeTables = (tablesJSON) => {
  // osquery JSON needs less parsing than it used to
  const parsedTables =
    typeof tablesJSON === "object" ? tablesJSON : JSON.parse(tablesJSON);
  return sortBy(parsedTables, (table) => {
    return table.name;
  });
};

export const osqueryTables = normalizeTables(osqueryTablesJSON);
export const osqueryTableNames = flatMap(osqueryTables, (table) => {
  return table.name;
});
