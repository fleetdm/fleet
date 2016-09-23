import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import ConnectedApp, { App } from './App';
import * as authActions from '../../redux/nodes/auth/actions';
import helpers from '../../test/helpers';
import local from '../../utilities/local';

describe('App - component', () => {
  const component = mount(<App />);

  afterEach(() => {
    restoreSpies();
    local.setItem('auth_token', null);
  });

  it('renders', () => {
    expect(component).toExist();
  });

  it('loads the current user if there is an auth token but no user', () => {
    local.setItem('auth_token', 'ABC123');

    const spy = spyOn(authActions, 'fetchCurrentUser').andCall(() => {
      return { type: 'LOAD_USER_ACTION' };
    });
    const store = { app: {}, auth: {} };
    const mockStore = helpers.reduxMockStore(store);
    const application = helpers.connectedComponent(ConnectedApp, { mockStore });

    mount(application);
    expect(spy).toHaveBeenCalled();
  });

  it('does not load the current user if is it already loaded', () => {
    local.setItem('auth_token', 'ABC123');

    const spy = spyOn(authActions, 'fetchCurrentUser').andCall(() => {
      return { type: 'LOAD_USER_ACTION' };
    });
    const store = {
      app: {},
      auth: {
        user: {
          id: 1,
          email: 'hi@thegnar.co',
        },
      },
    };
    const mockStore = helpers.reduxMockStore(store);
    const application = helpers.connectedComponent(ConnectedApp, { mockStore });

    mount(application);
    expect(spy).toNotHaveBeenCalled();
  });

  it('does not load the current user if there is no auth token', () => {
    local.setItem('auth_token', null);

    const spy = spyOn(authActions, 'fetchCurrentUser').andCall(() => {
      return { type: 'LOAD_USER_ACTION' };
    });
    const store = { app: {}, auth: {} };
    const mockStore = helpers.reduxMockStore(store);
    const application = helpers.connectedComponent(ConnectedApp, { mockStore });

    mount(application);
    expect(spy).toNotHaveBeenCalled();
  });

  it('renders the Style component', () => {
    expect(component.find('Style').length).toEqual(1);
  });

  it('renders the Footer component', () => {
    expect(component.find('Footer').length).toEqual(1);
  });
});
