import { find } from "lodash";

import { osqueryTables } from "utilities/osquery_tables";

export const SELECT_OSQUERY_TABLE = "SELECT_OSQUERY_TABLE";
export const defaultSelectedOsqueryTable = find(osqueryTables, {
  name: "users",
});
export const selectOsqueryTable = (tableName) => {
  const lowerTableName = tableName.toLowerCase();
  const selectedOsqueryTable = find(osqueryTables, { name: lowerTableName });

  return {
    type: SELECT_OSQUERY_TABLE,
    payload: { selectedOsqueryTable },
  };
};
