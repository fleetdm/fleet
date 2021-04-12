import { LOCATION_CHANGE } from "react-router-redux";

import { RENDER_FLASH, HIDE_FLASH } from "./actions";

export const initialState = {
  alertType: null,
  isVisible: false,
  message: null,
  undoAction: null,
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case RENDER_FLASH:
      return {
        alertType: payload.alertType,
        isVisible: true,
        message: payload.message,
        undoAction: payload.undoAction,
      };
    case HIDE_FLASH:
    case LOCATION_CHANGE:
      return {
        ...initialState,
      };
    default:
      return state;
  }
};
