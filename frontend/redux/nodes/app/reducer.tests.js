import expect from 'expect';
import reducer, { initialState } from './reducer';
import {
  configFailure,
  configSuccess,
  hideBackgroundImage,
  loadConfig,
  showBackgroundImage,
} from './actions';

describe('App - reducer', () => {
  it('sets the initial state', () => {
    expect(reducer(undefined, { type: 'SOME_ACTION' })).toEqual(initialState);
  });

  context('showBackgroundImage action', () => {
    it('shows the background image', () => {
      expect(reducer(initialState, showBackgroundImage)).toEqual({
        config: {},
        error: null,
        loading: false,
        showBackgroundImage: true,
        showRightSidePanel: false,
      });
    });
  });

  context('hideBackgroundImage action', () => {
    it('hides the background image', () => {
      const state = {
        ...initialState,
        showBackgroundImage: true,
        showRightSidePanel: false,
      };
      expect(reducer(state, hideBackgroundImage)).toEqual({
        config: {},
        error: null,
        loading: false,
        showBackgroundImage: false,
        showRightSidePanel: false,
      });
    });
  });

  context('loadConfig action', () => {
    it('sets the state to loading', () => {
      expect(reducer(initialState, loadConfig)).toEqual({
        config: {},
        error: null,
        loading: true,
        showBackgroundImage: false,
        showRightSidePanel: false,
      });
    });
  });

  context('configSuccess action', () => {
    it('sets the config in state', () => {
      const config = { name: 'Kolide' };
      const loadingConfigState = {
        ...initialState,
        loading: true,
      };
      expect(reducer(loadingConfigState, configSuccess(config))).toEqual({
        config,
        error: null,
        loading: false,
        showBackgroundImage: false,
        showRightSidePanel: false,
      });
    });
  });

  context('configFailure action', () => {
    it('sets the error in state', () => {
      const error = 'Unable to get config';
      const loadingConfigState = {
        ...initialState,
        loading: true,
      };
      expect(reducer(loadingConfigState, configFailure(error))).toEqual({
        config: {},
        error,
        loading: false,
        showBackgroundImage: false,
        showRightSidePanel: false,
      });
    });
  });
});
