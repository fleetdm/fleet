import { RESET_ERRORS } from "./actions";

const initialState = {
  errors: null,
};

const reducer = (state = initialState, { type, payload }) => {
  if (payload && payload.errors) {
    return {
      errors: payload.errors,
    };
  } else if (type === RESET_ERRORS) {
    return {
      errors: null,
    };
  }
  return state;
};

export default reducer;
