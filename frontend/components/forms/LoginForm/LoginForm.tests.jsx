import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import LoginForm from './LoginForm';
import { fillInFormInput } from '../../../utilities/testHelpers.js';

describe('LoginForm - component', () => {
  afterEach(restoreSpies);

  it('renders 2 InputFieldWithIcon components', () => {
    const form = mount(<LoginForm onSubmit={noop} />);

    expect(form.find('InputFieldWithIcon').length).toEqual(2);
  });

  it('updates component state when the email field is changed', () => {
    const form = mount(<LoginForm onSubmit={noop} />);

    const emailField = form.find({ name: 'email' });
    fillInFormInput(emailField, 'hello');

    const { formData } = form.state();
    expect(formData).toContain({
      email: 'hello',
    });
  });

  it('updates component state when the password field is changed', () => {
    const form = mount(<LoginForm onSubmit={noop} />);

    const passwordField = form.find({ name: 'password' });
    fillInFormInput(passwordField, 'hello');

    const { formData } = form.state();
    expect(formData).toContain({
      password: 'hello',
    });
  });

  it('it does not submit the form when the form fields have not been filled out', () => {
    const submitSpy = createSpy();
    const form = mount(<LoginForm onSubmit={submitSpy} />);
    const submitBtn = form.find('button');

    submitBtn.simulate('click');

    expect(submitBtn.prop('disabled')).toEqual(true);
    expect(submitSpy).toNotHaveBeenCalled();
  });

  it('submits the form data when the submit button is clicked', () => {
    const submitSpy = createSpy();
    const form = mount(<LoginForm onSubmit={submitSpy} />);
    const emailField = form.find({ name: 'email' });
    const passwordField = form.find({ name: 'password' });
    const submitBtn = form.find('button');

    fillInFormInput(emailField, 'my@email.com');
    fillInFormInput(passwordField, 'p@ssw0rd');
    submitBtn.simulate('click');

    expect(submitSpy).toHaveBeenCalledWith({
      email: 'my@email.com',
      password: 'p@ssw0rd',
    });
  });
});
