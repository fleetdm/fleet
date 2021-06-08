import reducer, { initialState } from "./reducer";
import {
  loadConfig,
  configFailure,
  configSuccess,
  loadEnrollSecret,
  enrollSecretFailure,
  enrollSecretSuccess,
  hideBackgroundImage,
  showBackgroundImage,
} from "./actions";

describe("App - reducer", () => {
  it("sets the initial state", () => {
    expect(reducer(undefined, { type: "SOME_ACTION" })).toEqual(initialState);
  });

  describe("showBackgroundImage action", () => {
    it("shows the background image", () => {
      expect(reducer(initialState, showBackgroundImage)).toEqual({
        ...initialState,
        showBackgroundImage: true,
      });
    });
  });

  describe("hideBackgroundImage action", () => {
    it("hides the background image", () => {
      const state = {
        ...initialState,
        showBackgroundImage: true,
      };
      expect(reducer(state, hideBackgroundImage)).toEqual({
        ...state,
        showBackgroundImage: false,
      });
    });
  });

  describe("loadConfig action", () => {
    it("sets the state to loading", () => {
      expect(reducer(initialState, loadConfig)).toEqual({
        ...initialState,
        loading: true,
      });
    });
  });

  describe("configSuccess action", () => {
    it("sets the config in state", () => {
      const config = { name: "Kolide" };
      const loadingConfigState = {
        ...initialState,
        loading: true,
      };
      expect(reducer(loadingConfigState, configSuccess(config))).toEqual({
        config,
        enrollSecret: [],
        error: {},
        loading: false,
        showBackgroundImage: false,
      });
    });
  });

  describe("configFailure action", () => {
    it("sets the error in state", () => {
      const error = "Unable to get config";
      const loadingConfigState = {
        ...initialState,
        loading: true,
      };
      expect(reducer(loadingConfigState, configFailure(error))).toEqual({
        config: {},
        enrollSecret: [],
        error,
        loading: false,
        showBackgroundImage: false,
      });
    });
  });

  describe("loadEnrollSecret action", () => {
    it("sets the state to loading", () => {
      expect(reducer(initialState, loadEnrollSecret)).toEqual({
        ...initialState,
        loading: true,
      });
    });
  });

  describe("enrollSecretSuccess action", () => {
    it("sets the enrollSecret in state", () => {
      const enrollSecret = [{ name: "Kolide" }];
      const loadingEnrollSecretState = {
        ...initialState,
        loading: true,
      };
      expect(
        reducer(loadingEnrollSecretState, enrollSecretSuccess(enrollSecret))
      ).toEqual({
        enrollSecret,
        config: {},
        error: {},
        loading: false,
        showBackgroundImage: false,
      });
    });
  });

  describe("enrollSecretFailure action", () => {
    it("sets the error in state", () => {
      const error = "Unable to get enrollSecret";
      const loadingEnrollSecretState = {
        ...initialState,
        loading: true,
      };
      expect(
        reducer(loadingEnrollSecretState, enrollSecretFailure(error))
      ).toEqual({
        enrollSecret: [],
        config: {},
        error,
        loading: false,
        showBackgroundImage: false,
      });
    });
  });
});
