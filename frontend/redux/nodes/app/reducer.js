import { HIDE_BACKGROUND_IMAGE, SHOW_BACKGROUND_IMAGE } from './actions';

const initialState = {
  showBackgroundImage: false,
};

const reducer = (state = initialState, action) => {
  switch (action.type) {
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
