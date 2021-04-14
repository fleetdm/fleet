import reducer, { initialState } from "./reducer";
import {
  loadOsqueryOptions,
  osqueryOptionsFailure,
  osqueryOptionsSuccess,
} from "./actions";

describe("Osquery - reducer", () => {
  it("sets the initial state", () => {
    expect(reducer(undefined, { type: "SOME_ACTION" })).toEqual(initialState);
  });

  it("sets the state to loading", () => {
    expect(reducer(initialState, loadOsqueryOptions)).toEqual({
      ...initialState,
      loading: true,
    });
  });

  it("sets the osquery options in state", () => {
    const osqueryOptions = { spec: {} };
    const loadingOsqueryOptionsState = {
      ...initialState,
      loading: true,
    };
    expect(
      reducer(loadingOsqueryOptionsState, osqueryOptionsSuccess(osqueryOptions))
    ).toEqual({
      loading: false,
      errors: {},
      options: osqueryOptions,
    });
  });

  it("sets errors in state", () => {
    const error = "Unable to get osquery options";
    const loadingOsqueryOptionsState = {
      ...initialState,
      loading: true,
    };
    expect(
      reducer(loadingOsqueryOptionsState, osqueryOptionsFailure(error))
    ).toEqual({
      loading: false,
      errors: error,
      options: {},
    });
  });
});
