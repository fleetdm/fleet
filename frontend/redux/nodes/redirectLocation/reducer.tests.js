import actions from "./actions";
import reducer, { initialState } from "./reducer";

describe("redirectLocation - reducer", () => {
  const redirectObject = { action: "PUSH", pathname: "admin" };
  const redirectAction = actions.setRedirectLocation(redirectObject);

  it("sets the initial state", () => {
    const newState = reducer(undefined, { type: "RANDOM_ACTION" });

    expect(newState).toEqual(initialState);
  });

  it("sets the redirect location in state", () => {
    const newState = reducer(initialState, redirectAction);

    expect(newState).toEqual(redirectObject);
  });

  it("clears the direction location in state", () => {
    const state = reducer(initialState, redirectAction);
    const newState = reducer(state, actions.clearRedirectLocation);

    expect(newState).toEqual(null);
  });
});
