// Action Types
const HIDE_PERSISTENT_FLASH = "HIDE_PERSISTENT_FLASH";
const SHOW_PERSISTENT_FLASH = "SHOW_PERSISTENT_FLASH";

export const ACTION_TYPES = {
  HIDE_PERSISTENT_FLASH,
  SHOW_PERSISTENT_FLASH,
};

// Actions
const hidePersistentFlash = { type: HIDE_PERSISTENT_FLASH };
const showPersistentFlash = (message) => {
  return {
    type: SHOW_PERSISTENT_FLASH,
    payload: { message },
  };
};

export default {
  hidePersistentFlash,
  showPersistentFlash,
};
