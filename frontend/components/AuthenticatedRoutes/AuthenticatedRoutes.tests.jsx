import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { Provider } from 'react-redux';
import AuthenticatedRoutes from './index';
import helpers from '../../test/helpers';

describe('AuthenticatedRoutes - component', () => {
  const renderedText = 'This text was rendered';
  const storeWithUser = {
    auth: {
      user: {
        id: 1,
        email: 'hi@thegnar.co',
      },
    },
  };
  const storeWithoutUser = { auth: {} };

  it('renders if there is a user in state', () => {
    const { reduxMockStore } = helpers;
    const mockStore = reduxMockStore(storeWithUser);
    const component = mount(
      <Provider store={mockStore}>
        <AuthenticatedRoutes>
          <div>{renderedText}</div>
        </AuthenticatedRoutes>
      </Provider>
    );

    expect(component.text()).toEqual(renderedText);
  });

  it('does not render without a user in state', () => {
    const { reduxMockStore } = helpers;
    const mockStore = reduxMockStore(storeWithoutUser);
    const component = mount(
      <Provider store={mockStore}>
        <AuthenticatedRoutes>
          <div>{renderedText}</div>
        </AuthenticatedRoutes>
      </Provider>
    );

    expect(component.html()).toNotExist();
  });
});

