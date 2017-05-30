import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import targetMock from 'test/target_mock';
import { noop } from 'lodash';
import DecoratorForm from './index';

describe('DecoratorForm - component', () => {
  beforeEach(targetMock);
  afterEach(restoreSpies);

  it('calls handle submit when validation passes', () => {
    const submitSpy = createSpy();
    const formData = {
      query: 'SELECT seconds FROM uptime;',
      type: 'interval',
      interval: 3600,
      built_in: false,
      name: 'Foo',
    };
    const form = mount(<DecoratorForm formData={formData} onTargetSelect={noop} handleSubmit={submitSpy} />);
    const submitButton = form.find('.decorator-form__form-btn--submit');
    submitButton.simulate('click');
    expect(submitSpy).toHaveBeenCalled();
    expect(form.state()).toInclude({
      errors: {},
      formData: { built_in: false, interval: 3600, name: 'Foo', query: 'SELECT seconds FROM uptime;', type: 'interval' },
    });
  });

  it('does not validate interval when decorator is load type', () => {
    const submitSpy = createSpy();
    const formData = {
      query: 'SELECT seconds FROM uptime;',
      type: 'load',
      interval: 3603,   // this will fail if validated
      built_in: false,
      name: 'Foo',
    };
    const form = mount(<DecoratorForm formData={formData} onTargetSelect={noop} handleSubmit={submitSpy} />);
    const submitButton = form.find('.decorator-form__form-btn--submit');
    submitButton.simulate('click');
    expect(submitSpy).toHaveBeenCalled();
    expect(form.state()).toInclude({
      errors: {},
      formData: { built_in: false, interval: 3603, name: 'Foo', query: 'SELECT seconds FROM uptime;', type: 'load' },
    });
  });

  it('validation fails when interval value not divisible by 60 for interval decorators', () => {
    const updateSpy = createSpy();
    const formData = {
      query: 'SELECT seconds FROM uptime;',
      type: 'interval',
      interval: 3601,
      built_in: false,
      name: 'Foo',
    };
    const form = mount(<DecoratorForm formData={formData} onTargetSelect={noop} onUpdate={updateSpy} />);
    const submitButton = form.find('.decorator-form__form-btn--submit');
    submitButton.simulate('click');
    expect(updateSpy).toNotHaveBeenCalled();
    expect(form.state()).toInclude({
      errors: {
        interval: 'Interval must be evenly divisible by 60',
        description: null,
      },
    });
  });

  it('validation fails for malformed sql statement', () => {
    const updateSpy = createSpy();
    const formData = {
      query: 'xxxxx seconds FROM uptime;',
      type: 'load',
      interval: 0,
      built_in: false,
      name: 'Foo',
    };
    const form = mount(<DecoratorForm formData={formData} onTargetSelect={noop} onUpdate={updateSpy} />);
    const submitButton = form.find('.decorator-form__form-btn--submit');
    submitButton.simulate('click');
    expect(updateSpy).toNotHaveBeenCalled();
    expect(form.state()).toInclude({
      errors: { query: 'Syntax error found near WITH Clause (Statement)' },
    });
  });
});
