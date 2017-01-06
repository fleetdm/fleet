import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { fillInFormInput } from 'test/helpers';
import PackForm from './index';

describe('PackForm - component', () => {
  afterEach(restoreSpies);

  it('renders the base error', () => {
    const baseError = 'Pack already exists';
    const formWithError = mount(<PackForm serverErrors={{ base: baseError }} handleSubmit={noop} />);
    const formWithoutError = mount(<PackForm handleSubmit={noop} />);

    expect(formWithError.text()).toInclude(baseError);
    expect(formWithoutError.text()).toNotInclude(baseError);
  });

  it('renders the correct components', () => {
    const form = mount(<PackForm />);

    expect(form.find('InputField').length).toEqual(2);
    expect(form.find('SelectTargetsDropdown').length).toEqual(1);
    expect(form.find('Button').length).toEqual(1);
  });

  it('validates the query pack name field', () => {
    const handleSubmitSpy = createSpy();
    const form = mount(<PackForm handleSubmit={handleSubmitSpy} />);

    form.find('form').simulate('submit');

    expect(handleSubmitSpy).toNotHaveBeenCalled();

    const formFieldProps = form.find('PackForm').prop('fields');

    expect(formFieldProps.name).toInclude({
      error: 'Title field must be completed',
    });
  });

  it('calls the handleSubmit prop when a valid form is submitted', () => {
    const handleSubmitSpy = createSpy();
    const form = mount(<PackForm handleSubmit={handleSubmitSpy} />).find('form');
    const nameField = form.find('InputField').find({ name: 'name' });

    fillInFormInput(nameField, 'Mac OS Attacks');

    form.simulate('submit');

    expect(handleSubmitSpy).toHaveBeenCalled();
  });
});
