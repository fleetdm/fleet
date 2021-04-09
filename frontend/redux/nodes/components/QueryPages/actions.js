import { find } from "lodash";

import { osqueryTables } from "utilities/osquery_tables";

export const SELECT_OSQUERY_TABLE = "SELECT_OSQUERY_TABLE";
export const SET_QUERY_TEXT = "SET_QUERY_TEXT";
export const SET_SELECTED_TARGETS = "SET_SELECTED_TARGETS";
export const SET_SELECTED_TARGETS_QUERY = "SET_SELECTED_TARGETS_QUERY";
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
export const setQueryText = (queryText) => {
  return {
    type: SET_QUERY_TEXT,
    payload: { queryText },
  };
};
export const setSelectedTargets = (selectedTargets) => {
  return {
    type: SET_SELECTED_TARGETS,
    payload: { selectedTargets },
  };
};
export const setSelectedTargetsQuery = (selectedTargetsQuery) => {
  return {
    type: SET_SELECTED_TARGETS_QUERY,
    payload: { selectedTargetsQuery },
  };
};
