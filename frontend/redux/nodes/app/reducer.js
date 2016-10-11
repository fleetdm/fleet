import {
  CONFIG_FAILURE,
  CONFIG_START,
  CONFIG_SUCCESS,
  HIDE_BACKGROUND_IMAGE,
  REMOVE_RIGHT_SIDE_PANEL,
  SHOW_BACKGROUND_IMAGE,
  SHOW_RIGHT_SIDE_PANEL,
} from './actions';

export const initialState = {
  config: {},
  error: null,
  loading: false,
  showBackgroundImage: false,
  showRightSidePanel: false,
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
        error: null,
        loading: false,
      };
    case CONFIG_FAILURE:
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
    case REMOVE_RIGHT_SIDE_PANEL:
      return {
        ...state,
        showRightSidePanel: false,
      };
    case SHOW_BACKGROUND_IMAGE:
      return {
        ...state,
        showBackgroundImage: true,
      };
    case SHOW_RIGHT_SIDE_PANEL:
      return {
        ...state,
        showRightSidePanel: true,
      };
    default:
      return state;
  }
};

export default reducer;
