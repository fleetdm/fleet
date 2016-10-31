import React from 'react';
import configureStore from 'redux-mock-store';
import { noop } from 'lodash';
import { Provider } from 'react-redux';
import { spyOn } from 'expect';
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

export const createAceSpy = () => {
  return spyOn(global.window.ace, 'edit').andReturn({
    $options: {},
    getValue: () => { return 'Hello world'; },
    getSession: () => {
      return {
        getMarkers: noop,
        setAnnotations: noop,
        setMode: noop,
        setUseWrapMode: noop,
      };
    },
    handleOptions: noop,
    handleMarkers: noop,
    on: noop,
    renderer: {
      setShowGutter: noop,
    },
    session: {
      on: noop,
      selection: {
        fromJSON: noop,
        toJSON: noop,
      },
    },
    setFontSize: noop,
    setMode: noop,
    setOption: noop,
    setOptions: noop,
    setShowPrintMargin: noop,
    setTheme: noop,
    setValue: noop,
  });
};

export default {
  connectedComponent,
  createAceSpy,
  fillInFormInput,
  reduxMockStore,
};

