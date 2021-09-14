import {
  CLEAR_AUTH_ERRORS,
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
  SSO_REDIRECT_REQUEST,
  SSO_REDIRECT_SUCCESS,
  SSO_REDIRECT_FAILURE,
  SSO_SETTINGS_REQUEST,
  SSO_SETTINGS_SUCCESS,
  SSO_SETTINGS_FAILURE,
} from "./actions";

export const initialState = {
  loading: false,
  errors: {},
  user: null,
  ssoRedirectURL: "",
  ssoSettings: {},
};

const reducer = (state = initialState, action) => {
  switch (action.type) {
    case CLEAR_AUTH_ERRORS:
      return {
        ...state,
        errors: {},
      };
    case LOGIN_REQUEST:
    case LOGOUT_REQUEST:
    case UPDATE_USER_REQUEST:
    case SSO_REDIRECT_REQUEST:
    case SSO_SETTINGS_REQUEST:
      return {
        ...state,
        loading: true,
      };
    case SSO_SETTINGS_SUCCESS:
      return {
        ...state,
        loading: false,
        ssoSettings: action.payload.ssoSettings,
      };
    case LOGIN_SUCCESS:
      return {
        ...state,
        loading: false,
        user: action.payload.user,
      };
    case SSO_REDIRECT_SUCCESS:
      return {
        ...state,
        loading: false,
        ssoRedirectURL: action.payload.ssoRedirectURL,
      };
    case SSO_REDIRECT_FAILURE:
    case SSO_SETTINGS_FAILURE:
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
        isLoggingOut: true, // TODO: temporary until redux is removed - 9/1/21 MP
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
