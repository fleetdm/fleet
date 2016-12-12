import { SET_DISPLAY } from './actions';

export const initialState = {
  display: 'Grid',
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case SET_DISPLAY:
      return {
        ...state,
        display: payload.display,
      };
    default:
      return state;
  }
};
