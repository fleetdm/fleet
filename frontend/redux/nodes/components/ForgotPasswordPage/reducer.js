import {
  CLEAR_FORGOT_PASSWORD_ERRORS,
  FORGOT_PASSWORD_ERROR,
  FORGOT_PASSWORD_REQUEST,
  FORGOT_PASSWORD_SUCCESS,
} from "./actions";

export const initialState = {
  email: null,
  errors: {},
  loading: false,
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case CLEAR_FORGOT_PASSWORD_ERRORS:
      return {
        ...state,
        errors: {},
      };
    case FORGOT_PASSWORD_REQUEST:
      return {
        ...state,
        errors: {},
        loading: true,
      };
    case FORGOT_PASSWORD_SUCCESS:
      return {
        ...state,
        email: payload.data.email,
        errors: {},
        loading: false,
      };
    case FORGOT_PASSWORD_ERROR:
      return {
        ...state,
        email: null,
        errors: payload.errors,
        loading: false,
      };
    default:
      return state;
  }
};

export default reducer;
