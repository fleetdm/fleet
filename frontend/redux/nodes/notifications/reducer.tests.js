import { LOCATION_CHANGE } from "react-router-redux";

import reducer, { initialState } from "./reducer";
import { hideFlash, renderFlash } from "./actions";

describe("Notifications - reducer", () => {
  it("Updates state with notification info when RENDER_FLASH is dispatched", () => {
    const undoAction = { type: "UNDO" };
    const newState = reducer(
      initialState,
      renderFlash("success", "You did it!", undoAction)
    );

    expect(newState).toEqual({
      alertType: "success",
      isVisible: true,
      message: "You did it!",
      undoAction,
    });
  });

  it("Updates state to hide notifications when HIDE_FLASH is dispatched", () => {
    const stateWithFlashDisplayed = reducer(
      initialState,
      renderFlash("success", "You did it!")
    );
    const newState = reducer(stateWithFlashDisplayed, hideFlash);

    expect(newState).toEqual({
      alertType: null,
      isVisible: false,
      message: null,
      undoAction: null,
    });
  });

  it("Updates state to hide notifications during location change", () => {
    const stateWithFlashDisplayed = reducer(
      initialState,
      renderFlash("success", "You did it!")
    );
    const newState = reducer(stateWithFlashDisplayed, {
      type: LOCATION_CHANGE,
    });

    expect(newState).toEqual({
      alertType: null,
      isVisible: false,
      message: null,
      undoAction: null,
    });
  });
});
