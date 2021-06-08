import {
  CONFIG_FAILURE,
  CONFIG_START,
  CONFIG_SUCCESS,
  ENROLL_SECRET_FAILURE,
  ENROLL_SECRET_START,
  ENROLL_SECRET_SUCCESS,
  HIDE_BACKGROUND_IMAGE,
  SHOW_BACKGROUND_IMAGE,
} from "./actions";

export const initialState = {
  config: {},
  enrollSecret: [],
  error: {},
  loading: false,
  showBackgroundImage: false,
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case CONFIG_START:
      return {
        ...state,
        loading: true,
      };
    case CONFIG_SUCCESS:
      return {
        ...state,
        config: payload.data,
        error: {},
        loading: false,
      };
    case CONFIG_FAILURE:
      return {
        ...state,
        error: payload.error,
        loading: false,
      };
    case ENROLL_SECRET_START:
      return {
        ...state,
        loading: true,
      };
    case ENROLL_SECRET_SUCCESS:
      return {
        ...state,
        enrollSecret: payload.data,
        error: {},
        loading: false,
      };
    case ENROLL_SECRET_FAILURE:
      return {
        ...state,
        error: payload.error,
        loading: false,
      };
    case HIDE_BACKGROUND_IMAGE:
      return {
        ...state,
        showBackgroundImage: false,
      };
    case SHOW_BACKGROUND_IMAGE:
      return {
        ...state,
        showBackgroundImage: true,
      };
    default:
      return state;
  }
};

export default reducer;
