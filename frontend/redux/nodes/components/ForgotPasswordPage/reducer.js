import {
  CLEAR_FORGOT_PASSWORD_ERRORS,
  FORGOT_PASSWORD_ERROR,
  FORGOT_PASSWORD_REQUEST,
  FORGOT_PASSWORD_SUCCESS,
} from './actions';

export const initialState = {
  email: null,
  error: null,
  loading: false,
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case CLEAR_FORGOT_PASSWORD_ERRORS:
      return {
        ...state,
        error: null,
      };
    case FORGOT_PASSWORD_REQUEST:
      return {
        ...state,
        loading: true,
      };
    case FORGOT_PASSWORD_SUCCESS:
      return {
        ...state,
        email: payload.data.email,
        error: null,
        loading: false,
      };
    case FORGOT_PASSWORD_ERROR:
      return {
        ...state,
        email: null,
        error: payload.error,
        loading: false,
      };
    default:
      return state;
  }
};

export default reducer;
