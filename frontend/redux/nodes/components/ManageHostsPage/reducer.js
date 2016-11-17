import { SET_SELECTED_LABEL } from './actions';

const initialState = {
  selectedLabel: null,
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case SET_SELECTED_LABEL:
      return {
        ...state,
        selectedLabel: payload.selectedLabel,
      };
    default:
      return state;
  }
};
