import { ACTION_TYPES } from "redux/nodes/persistent_flash/actions";

export const initialState = {
  showFlash: false,
  message: "",
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case ACTION_TYPES.HIDE_PERSISTENT_FLASH:
      return {
        showFlash: false,
        message: "",
      };
    case ACTION_TYPES.SHOW_PERSISTENT_FLASH:
      return {
        showFlash: true,
        message: payload.message,
      };
    default:
      return state;
  }
};
