export const RENDER_FLASH = "RENDER_FLASH";
export const HIDE_FLASH = "HIDE_FLASH";

export const renderFlash = (alertType, message, undoAction) => {
  return {
    type: RENDER_FLASH,
    payload: {
      alertType,
      message,
      undoAction,
    },
  };
};

export const hideFlash = { type: HIDE_FLASH };
