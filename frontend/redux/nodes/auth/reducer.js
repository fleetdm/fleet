import {
  CLEAR_AUTH_ERRORS,
  LICENSE_FAILURE,
  LICENSE_REQUEST,
  LICENSE_SUCCESS,
  LOGIN_FAILURE,
  LOGIN_REQUEST,
  LOGIN_SUCCESS,
  UPDATE_USER_FAILURE,
  UPDATE_USER_REQUEST,
  UPDATE_USER_SUCCESS,
  LOGOUT_FAILURE,
  LOGOUT_REQUEST,
  LOGOUT_SUCCESS,
  PERFORM_REQUIRED_PASSWORD_RESET_REQUEST,
  PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
  PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
} from './actions';

export const initialState = {
  license: {},
  loading: false,
  errors: {},
  user: null,
};

const reducer = (state = initialState, action) => {
  switch (action.type) {
    case CLEAR_AUTH_ERRORS:
      return {
        ...state,
        errors: {},
      };
    case LICENSE_REQUEST:
    case LOGIN_REQUEST:
    case LOGOUT_REQUEST:
    case UPDATE_USER_REQUEST:
      return {
        ...state,
        loading: true,
      };
    case LICENSE_SUCCESS:
      return {
        ...state,
        loading: false,
        license: action.payload.license,
      };
    case LOGIN_SUCCESS:
      return {
        ...state,
        loading: false,
        user: action.payload.user,
      };
    case LICENSE_FAILURE:
    case LOGIN_FAILURE:
      return {
        ...state,
        loading: false,
        errors: action.payload.errors,
      };
    case UPDATE_USER_SUCCESS:
      return {
        ...state,
        loading: false,
        user: action.payload.user,
      };
    case UPDATE_USER_FAILURE:
      return {
        ...state,
        loading: false,
        errors: action.payload.errors,
      };
    case LOGOUT_SUCCESS:
      return {
        ...state,
        loading: false,
        user: null,
      };
    case LOGOUT_FAILURE:
      return {
        ...state,
        loading: false,
        errors: action.payload.errors,
      };
    case PERFORM_REQUIRED_PASSWORD_RESET_REQUEST:
      return {
        ...state,
        errors: {},
        loading: true,
      };
    case PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS:
      return {
        ...state,
        errors: {},
        loading: false,
        user: action.payload.user,
      };
    case PERFORM_REQUIRED_PASSWORD_RESET_FAILURE:
      return {
        ...state,
        loading: false,
        errors: action.payload.errors,
      };
    default:
      return state;
  }
};

export default reducer;
