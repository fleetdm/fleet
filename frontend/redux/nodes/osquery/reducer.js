import {
  OSQUERY_OPTIONS_FAILURE,
  OSQUERY_OPTIONS_START,
  OSQUERY_OPTIONS_SUCCESS,
} from "./actions";

export const initialState = {
  options: {},
  errors: {},
  loading: false,
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case OSQUERY_OPTIONS_START:
      return {
        ...state,
        loading: true,
      };
    case OSQUERY_OPTIONS_SUCCESS:
      return {
        ...state,
        options: payload.data,
        loading: false,
      };
    case OSQUERY_OPTIONS_FAILURE:
      return {
        ...state,
        errors: payload.errors,
        loading: false,
      };
    default:
      return state;
  }
};

export default reducer;
