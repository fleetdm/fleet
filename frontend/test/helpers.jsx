import React from 'react';
import configureStore from 'redux-mock-store';
import { Provider } from 'react-redux';
import thunk from 'redux-thunk';

export const fillInFormInput = (inputComponent, value) => {
  return inputComponent.simulate('change', { target: { value } });
};

export const reduxMockStore = (store = {}) => {
  const middlewares = [thunk];
  const mockStore = configureStore(middlewares);

  return mockStore(store);
};

export const connectedComponent = (ComponentClass, { props = {}, mockStore }) => {
  return (
    <Provider store={mockStore}>
      <ComponentClass {...props} />
    </Provider>
  );
};

export default {
  connectedComponent,
  fillInFormInput,
  reduxMockStore,
};

