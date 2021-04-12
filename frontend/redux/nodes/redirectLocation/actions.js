export const CLEAR_REDIRECT_LOCATION = "CLEAR_REDIRECT_LOCATION";
export const SET_REDIRECT_LOCATION = "SET_REDIRECT_LOCATION";

export const clearRedirectLocation = { type: CLEAR_REDIRECT_LOCATION };
export const setRedirectLocation = (redirectLocation) => {
  return {
    type: SET_REDIRECT_LOCATION,
    payload: {
      redirectLocation,
    },
  };
};

export default {
  CLEAR_REDIRECT_LOCATION,
  clearRedirectLocation,
  SET_REDIRECT_LOCATION,
  setRedirectLocation,
};
