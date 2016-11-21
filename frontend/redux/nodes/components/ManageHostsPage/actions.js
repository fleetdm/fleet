export const SET_DISPLAY = 'SET_DISPLAY';
export const setDisplay = (display) => {
  return {
    type: SET_DISPLAY,
    payload: {
      display,
    },
  };
};

export const SET_SELECTED_LABEL = 'SET_SELECTED_LABEL';
export const setSelectedLabel = (selectedLabel) => {
  return {
    type: SET_SELECTED_LABEL,
    payload: {
      selectedLabel,
    },
  };
};

export default {
  setDisplay,
  setSelectedLabel,
};
