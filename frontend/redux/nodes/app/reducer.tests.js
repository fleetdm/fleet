import expect from 'expect';

import reducer, { initialState } from './reducer';
import {
  configFailure,
  configSuccess,
  hideBackgroundImage,
  showBackgroundImage,
  toggleSmallNav,
  loadConfig,
} from './actions';

describe('App - reducer', () => {
  it('sets the initial state', () => {
    expect(reducer(undefined, { type: 'SOME_ACTION' })).toEqual(initialState);
  });

  context('toggleSmallNav action', () => {
    it('toggles isSmallNav on', () => {
      expect(reducer(initialState, toggleSmallNav)).toEqual({
        ...initialState,
        isSmallNav: true,
      });
    });

    it('toggles isSmallNav off', () => {
      const state = {
        ...initialState,
        isSmallNav: true,
      };

      expect(reducer(state, toggleSmallNav)).toEqual({
        ...state,
        isSmallNav: false,
      });
    });
  });

  context('showBackgroundImage action', () => {
    it('shows the background image', () => {
      expect(reducer(initialState, showBackgroundImage)).toEqual({
        ...initialState,
        showBackgroundImage: true,
      });
    });
  });

  context('hideBackgroundImage action', () => {
    it('hides the background image', () => {
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

  context('loadConfig action', () => {
    it('sets the state to loading', () => {
      expect(reducer(initialState, loadConfig)).toEqual({
        ...initialState,
        loading: true,
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
        error: {},
        loading: false,
        isSmallNav: false,
        showBackgroundImage: false,
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
        isSmallNav: false,
        showBackgroundImage: false,
      });
    });
  });
});
