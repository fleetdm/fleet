import {
  defaultSelectedOsqueryTable,
  SELECT_OSQUERY_TABLE,
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
    default:
      return state;
  }
};

export default reducer;
