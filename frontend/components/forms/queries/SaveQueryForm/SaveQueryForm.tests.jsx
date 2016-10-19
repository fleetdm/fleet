import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import helpers from '../../../../test/helpers';
import SaveQueryForm from './index';

const { fillInFormInput } = helpers;
const queryName = 'My New Query';

describe('SaveQueryForm - component', () => {
  afterEach(restoreSpies);

  it('handles query name input changes', () => {
    const form = mount(
      <SaveQueryForm onSubmit={noop} saveQuery />
    );
    const queryNameInput = form.find({ name: 'name' });

    fillInFormInput(queryNameInput, queryName);

    const { formData } = form.state();

    expect(formData).toEqual({
      description: null,
      duration: 'short',
      hosts: 'all',
      hostsPercentage: null,
      name: queryName,
      platforms: 'all',
      scanInterval: 0,
    });
  });

  it('does not submit the form if it is invalid', () => {
    const onSubmit = createSpy();
    const form = mount(
      <SaveQueryForm onSubmit={onSubmit} saveQuery />
    );

    form.simulate('submit');

    expect(onSubmit).toNotHaveBeenCalled();
  });

  it('calls onSubmit with the formData and "RUN_AND_SAVE" runType when the saveQuery prop is present', () => {
    const onSubmit = createSpy();
    const form = mount(
      <SaveQueryForm onSubmit={onSubmit} saveQuery />
    );
    const queryNameInput = form.find({ name: 'name' });

    fillInFormInput(queryNameInput, queryName);
    form.simulate('submit');

    expect(onSubmit).toHaveBeenCalledWith({
      runType: 'RUN_AND_SAVE',
      formData: {
        description: null,
        duration: 'short',
        hosts: 'all',
        hostsPercentage: null,
        name: queryName,
        platforms: 'all',
        scanInterval: 0,
      },
    });
  });

  it('calls onSubmit with the formData and "RUN" runType without the saveQuery prop', () => {
    const onSubmit = createSpy();
    const form = mount(
      <SaveQueryForm onSubmit={onSubmit} />
    );

    form.simulate('submit');

    expect(onSubmit).toHaveBeenCalledWith({
      runType: 'RUN',
      formData: {
        description: null,
        duration: 'short',
        hosts: 'all',
        hostsPercentage: null,
        name: null,
        platforms: 'all',
        scanInterval: 0,
      },
    });
  });
});
