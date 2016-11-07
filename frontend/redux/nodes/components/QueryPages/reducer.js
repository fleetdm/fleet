import {
  defaultSelectedOsqueryTable,
  SELECT_OSQUERY_TABLE,
  SET_QUERY_TEXT,
  SET_SELECTED_TARGETS,
} from './actions';

export const initialState = {
  queryText: 'SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid',
  selectedOsqueryTable: defaultSelectedOsqueryTable,
  selectedTargets: [],
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case SELECT_OSQUERY_TABLE:
      return {
        ...state,
        selectedOsqueryTable: payload.selectedOsqueryTable,
      };
    case SET_QUERY_TEXT:
      return {
        ...state,
        queryText: payload.queryText,
      };
    case SET_SELECTED_TARGETS:
      return {
        ...state,
        selectedTargets: payload.selectedTargets,
      };
    default:
      return state;
  }
};

export default reducer;
