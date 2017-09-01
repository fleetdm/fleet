import React from 'react';
import configureStore from 'redux-mock-store';
import expect, { spyOn } from 'expect';
import { noop } from 'lodash';
import { Provider } from 'react-redux';
import thunk from 'redux-thunk';

import authMiddleware from 'redux/middlewares/auth';
import redirectMiddleware from 'redux/middlewares/redirect';

export const fillInFormInput = (inputComponent, value) => {
  return inputComponent.simulate('change', { target: { value } });
};

export const reduxMockStore = (store = {}) => {
  const middlewares = [thunk, authMiddleware, redirectMiddleware];
  const mockStore = configureStore(middlewares);

  return mockStore(store);
};

export const connectedComponent = (ComponentClass, options = {}) => {
  const { mockStore = reduxMockStore(), props = {} } = options;

  return (
    <Provider store={mockStore}>
      <ComponentClass {...props} />
    </Provider>
  );
};

export const itBehavesLikeAFormDropdownElement = (form, inputName) => {
  const dropdownField = form.find('Select').findWhere(s => s.prop('name') === `${inputName}-select`);

  expect(dropdownField.length).toEqual(1);

  const inputNode = dropdownField.find('input');

  const options = dropdownField.prop('options');

  fillInFormInput(inputNode, options[0].label);
  dropdownField.find('.Select-option').first().simulate('mousedown');

  expect(form.state().formData).toInclude({ [inputName]: options[0].value });
};

export const itBehavesLikeAFormInputElement = (form, inputName, inputType = 'InputField', inputText = 'some text') => {
  const Input = form.find({ name: inputName });
  const inputField = inputType === 'textarea' ? Input.find('textarea') : Input.find('input');

  expect(inputField.length).toEqual(1);

  if (inputType === 'Checkbox') {
    const inputValue = form.state().formData[inputName];

    inputField.simulate('change');

    expect(form.state().formData[inputName]).toEqual(!inputValue);
  } else {
    fillInFormInput(inputField, inputText);

    expect(form.state().formData).toInclude({ [inputName]: inputText });
  }
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

export const stubbedOsqueryTable = {
  attributes: {},
  blacklisted: false,
  columns: [
    { description: 'User ID', name: 'uid', options: { index: true }, type: 'BIGINT_TYPE' },
    { description: 'Group ID (unsigned)', name: 'gid', options: {}, type: 'BIGINT_TYPE' },
    { description: 'User ID as int64 signed (Apple)', name: 'uid_signed', options: {}, type: 'BIGINT_TYPE' },
    { description: 'Default group ID as int64 signed (Apple)', name: 'gid_signed', options: {}, type: 'BIGINT_TYPE' },
    { description: 'Username', name: 'username', options: {}, type: 'TEXT_TYPE' },
    { description: 'Optional user description', name: 'description', options: {}, type: 'TEXT_TYPE' },
    { description: "User's home directory", name: 'directory', options: {}, type: 'TEXT_TYPE' },
    { description: "User's configured default shell", name: 'shell', options: {}, type: 'TEXT_TYPE' },
    { description: "User's UUID (Apple)", name: 'uuid', options: {}, type: 'TEXT_TYPE' },
  ],
  description: 'Local system users.',
  examples: [
    'select * from users where uid = 1000',
    "select * from users where username = 'root'",
    'select count(*) from users u, user_groups ug where u.uid = ug.uid',
  ],
  foreign_keys: [],
  function: 'genUsers',
  name: 'users',
  profile: {},
};

export default {
  connectedComponent,
  createAceSpy,
  fillInFormInput,
  itBehavesLikeAFormInputElement,
  reduxMockStore,
  stubbedOsqueryTable,
};

