export const SET_DISPLAY = 'SET_DISPLAY';
export const setDisplay = (display) => {
  return {
    type: SET_DISPLAY,
    payload: {
      display,
    },
  };
};

export default { setDisplay };
