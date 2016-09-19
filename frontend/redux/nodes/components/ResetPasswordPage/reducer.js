import {
  CLEAR_RESET_PASSWORD_ERRORS,
  RESET_PASSWORD_ERROR,
  RESET_PASSWORD_REQUEST,
  RESET_PASSWORD_SUCCESS,
} from './actions';

export const initialState = {
  error: null,
  loading: false,
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case CLEAR_RESET_PASSWORD_ERRORS:
      return {
        ...state,
        error: null,
      };
    case RESET_PASSWORD_ERROR:
      return {
        ...state,
        error: payload.error,
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
        error: null,
        loading: false,
      };
    default:
      return state;
  }
};

