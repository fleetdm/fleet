import {
  defaultSelectedOsqueryTable,
  SELECT_OSQUERY_TABLE,
  SET_QUERY_TEXT,
  SET_SELECTED_TARGETS,
  SET_SELECTED_TARGETS_QUERY,
} from "./actions";

export const initialState = {
  queryText: "SELECT * FROM osquery_info",
  selectedOsqueryTable: defaultSelectedOsqueryTable,
  selectedTargets: [],
  selectedTargetsQuery: "",
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
    case SET_SELECTED_TARGETS_QUERY:
      return {
        ...state,
        selectedTargetsQuery: payload.selectedTargetsQuery,
      };
    default:
      return state;
  }
};

export default reducer;
