import { CLEAR_REDIRECT_LOCATION, SET_REDIRECT_LOCATION } from "./actions";

export const initialState = null;

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case CLEAR_REDIRECT_LOCATION:
      return null;
    case SET_REDIRECT_LOCATION:
      return {
        ...payload.redirectLocation,
      };
    default:
      return state;
  }
};

export default reducer;
