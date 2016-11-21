import { SET_DISPLAY, SET_SELECTED_LABEL } from './actions';

export const initialState = {
  display: 'Grid',
  selectedLabel: null,
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case SET_DISPLAY:
      return {
        ...state,
        display: payload.display,
      };
    case SET_SELECTED_LABEL:
      return {
        ...state,
        selectedLabel: payload.selectedLabel,
      };
    default:
      return state;
  }
};
