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
  setSelectedLabel,
};
