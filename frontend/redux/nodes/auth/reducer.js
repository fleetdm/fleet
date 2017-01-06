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
} from './actions';

export const initialState = {
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
    case LOGIN_REQUEST:
    case LOGOUT_REQUEST:
    case UPDATE_USER_REQUEST:
      return {
        ...state,
        loading: true,
      };
    case LOGIN_SUCCESS:
      return {
        ...state,
        loading: false,
        user: action.payload.user,
      };
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
    default:
      return state;
  }
};

export default reducer;
