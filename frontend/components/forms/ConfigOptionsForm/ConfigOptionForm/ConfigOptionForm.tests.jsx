import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConfigOptionForm from 'components/forms/ConfigOptionsForm/ConfigOptionForm';
import {
  itBehavesLikeAFormInputElement,
  itBehavesLikeAFormDropdownElement,
} from 'test/helpers';

describe('ConfigOptionForm - form', () => {
  afterEach(restoreSpies);

  it('renders form fields for the config option name and value', () => {
    const configNameOptions = [{ label: 'My option', value: 'my_option' }];
    const form = mount(
      <ConfigOptionForm
        configNameOptions={configNameOptions}
        handleSubmit={noop}
        onRemove={noop}
      />
    );

    itBehavesLikeAFormDropdownElement(form, 'name');
    itBehavesLikeAFormInputElement(form, 'value');
  });

  it('calls the onChangeFunc prop when the form updates', () => {
    const spy = createSpy();
    const configNameOptions = [{ label: 'My option', value: 'my_option' }];
    const form = mount(
      <ConfigOptionForm
        configNameOptions={configNameOptions}
        handleSubmit={noop}
        onChangeFunc={spy}
        onRemove={noop}
      />
    );

    itBehavesLikeAFormInputElement(form, 'value', 'InputField', 'new config option value');
    itBehavesLikeAFormDropdownElement(form, 'name');

    expect(spy).toHaveBeenCalledWith('value', 'new config option value');
    expect(spy).toHaveBeenCalledWith('name', 'my_option');
  });

  it('renders the input fields as disabled when the option is read_only or name and value are present', () => {
    const formData = { name: 'My option', value: 'My value', read_only: false };
    const configNameOptions = [formData];
    const disabledForm = mount(
      <ConfigOptionForm
        configNameOptions={configNameOptions}
        formData={formData}
        handleSubmit={noop}
        onRemove={noop}
      />
    );
    const enabledForm = mount(
      <ConfigOptionForm
        configNameOptions={configNameOptions}
        formData={{ ...formData, value: null }}
        handleSubmit={noop}
        onRemove={noop}
      />
    );
    const readOnlyForm = mount(
      <ConfigOptionForm
        configNameOptions={configNameOptions}
        formData={{ ...formData, read_only: true }}
        handleSubmit={noop}
        onRemove={noop}
      />
    );

    const disabledNameField = disabledForm.find('Dropdown');
    const disabledValueField = disabledForm.find({ name: 'value' });
    const enabledNameField = enabledForm.find('Dropdown');
    const enabledValueField = enabledForm.find({ name: 'value' });
    const readOnlyNameField = readOnlyForm.find('Dropdown');
    const readOnlyValueField = readOnlyForm.find({ name: 'value' });

    expect(disabledNameField.prop('disabled')).toEqual(true);
    expect(disabledValueField.prop('disabled')).toEqual(false);
    expect(enabledNameField.prop('disabled')).toEqual(false);
    expect(enabledValueField.prop('disabled')).toEqual(false);
    expect(readOnlyNameField.prop('disabled')).toEqual(true);
    expect(readOnlyValueField.prop('disabled')).toEqual(true);
  });

  it('calls onRemove with the formdata when the ex icon is clicked', () => {
    const formData = { name: 'My option', value: 'my_option', read_only: false };
    const configNameOptions = [formData];
    const spy = createSpy();
    const form = mount(
      <ConfigOptionForm
        configNameOptions={configNameOptions}
        formData={formData}
        handleSubmit={noop}
        onRemove={spy}
      />
    );
    const exIcon = form.find('Button');

    exIcon.simulate('click');

    expect(spy).toHaveBeenCalledWith(formData);
  });
});

