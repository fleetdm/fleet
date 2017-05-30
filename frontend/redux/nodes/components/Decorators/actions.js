import { find } from 'lodash';

import { osqueryTables } from 'utilities/osquery_tables';

export const SELECT_DECORATOR_TABLE = 'SELECT_DECORATOR_TABLE';
export const SET_DECORATOR_QUERY_TEXT = 'SET_DECORATOR_QUERY_TEXT';
export const SET_SELECTED_DECORATOR_TARGETS = 'SET_SELECTED_DECORATOR_TARGETS';
export const SET_SELECTED_DECORATOR_TARGETS_QUERY = 'SET_SELECTED_DECORATOR_TARGETS_QUERY';
export const defaultSelectedOsqueryTable = find(osqueryTables, { name: 'uptime' });
export const selectOsqueryTable = (tableName) => {
  const lowerTableName = tableName.toLowerCase();
  const selectedOsqueryTable = find(osqueryTables, { name: lowerTableName });

  return {
    type: SELECT_DECORATOR_TABLE,
    payload: { selectedOsqueryTable },
  };
};
export const setQueryText = (queryText) => {
  return {
    type: SET_DECORATOR_QUERY_TEXT,
    payload: { queryText },
  };
};
export const setSelectedTargets = (selectedTargets) => {
  return {
    type: SET_SELECTED_DECORATOR_TARGETS,
    payload: { selectedTargets },
  };
};
export const setSelectedTargetsQuery = (selectedTargetsQuery) => {
  return {
    type: SET_SELECTED_DECORATOR_TARGETS_QUERY,
    payload: { selectedTargetsQuery },
  };
};
