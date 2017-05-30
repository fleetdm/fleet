import {
  defaultSelectedOsqueryTable,
  SELECT_DECORATOR_TABLE,
  SET_DECORATOR_QUERY_TEXT,
  SET_SELECTED_DECORATOR_TARGETS,
  SET_SELECTED_DECORATOR_TARGETS_QUERY,
} from './actions';

export const initialState = {
  queryText: 'SELECT total_seconds AS uptime FROM uptime',
  selectedOsqueryTable: defaultSelectedOsqueryTable,
  selectedTargets: [],
  selectedTargetsQuery: '',
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case SELECT_DECORATOR_TABLE:
      return {
        ...state,
        selectedOsqueryTable: payload.selectedOsqueryTable,
      };
    case SET_DECORATOR_QUERY_TEXT:
      return {
        ...state,
        queryText: payload.queryText,
      };
    case SET_SELECTED_DECORATOR_TARGETS:
      return {
        ...state,
        selectedTargets: payload.selectedTargets,
      };
    case SET_SELECTED_DECORATOR_TARGETS_QUERY:
      return {
        ...state,
        selectedTargetsQuery: payload.selectedTargetsQuery,
      };
    default:
      return state;
  }
};

export default reducer;
