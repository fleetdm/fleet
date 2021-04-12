import actions from "redux/nodes/persistent_flash/actions";
import reducer, { initialState } from "redux/nodes/persistent_flash/reducer";

describe("persistent_flash - reducer", () => {
  it("sets the initial state", () => {
    const nextState = reducer(undefined, { type: "SOME_ACTION" });

    expect(nextState).toEqual(initialState);
  });

  it("shows the flash and sets the message when showPersistentFlash is dispatched", () => {
    const message = "This is the flash message";
    const action = actions.showPersistentFlash(message);

    expect(reducer(initialState, action)).toEqual({
      showFlash: true,
      message,
    });
  });

  it("hides the flash and removes the message when hidePersistentFlash is dispatched", () => {
    const currentState = { showFlash: true, message: "something" };
    const action = actions.hidePersistentFlash;

    expect(reducer(currentState, action)).toEqual({
      showFlash: false,
      message: "",
    });
  });
});
