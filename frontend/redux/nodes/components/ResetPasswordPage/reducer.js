import {
  CLEAR_RESET_PASSWORD_ERRORS,
  RESET_PASSWORD_ERROR,
  RESET_PASSWORD_REQUEST,
  RESET_PASSWORD_SUCCESS,
} from "./actions";

export const initialState = {
  errors: {},
  loading: false,
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case CLEAR_RESET_PASSWORD_ERRORS:
      return {
        ...state,
        errors: {},
      };
    case RESET_PASSWORD_ERROR:
      return {
        ...state,
        errors: payload.errors,
        loading: false,
      };
    case RESET_PASSWORD_REQUEST:
      return {
        ...state,
        loading: true,
      };
    case RESET_PASSWORD_SUCCESS:
      return {
        ...state,
        errors: {},
        loading: false,
      };
    default:
      return state;
  }
};
