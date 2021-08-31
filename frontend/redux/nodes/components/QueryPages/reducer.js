import {
  defaultSelectedOsqueryTable,
  SELECT_OSQUERY_TABLE,
} from "./actions";

export const initialState = {
  selectedOsqueryTable: defaultSelectedOsqueryTable,
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
