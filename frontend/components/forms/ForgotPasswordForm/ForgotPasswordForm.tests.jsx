import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ForgotPasswordForm from './ForgotPasswordForm';
import { fillInFormInput } from '../../../test/helpers';

const email = 'hi@thegnar.co';

describe('ForgotPasswordForm - component', () => {
  afterEach(restoreSpies);

  it('renders the base error', () => {
    const baseError = 'Cant find the specified user';
    const formWithError = mount(<ForgotPasswordForm serverErrors={{ base: baseError }} handleSubmit={noop} />);
    const formWithoutError = mount(<ForgotPasswordForm handleSubmit={noop} />);

    expect(formWithError.text()).toInclude(baseError);
    expect(formWithoutError.text()).toNotInclude(baseError);
  });

  it('renders an InputFieldWithIcon components', () => {
    const form = mount(<ForgotPasswordForm handleSubmit={noop} />);

    expect(form.find('InputFieldWithIcon').length).toEqual(1);
  });

  it('updates component state when the email field is changed', () => {
    const form = mount(<ForgotPasswordForm handleSubmit={noop} />);

    const emailField = form.find({ name: 'email' });
    fillInFormInput(emailField, email);

    const { formData } = form.state();
    expect(formData).toContain({ email });
  });

  it('it does not submit the form when the form fields have not been filled out', () => {
    const submitSpy = createSpy();
    const form = mount(<ForgotPasswordForm handleSubmit={submitSpy} />);
    const submitBtn = form.find('button');

    submitBtn.simulate('submit');

    expect(form.state().errors).toInclude({
      email: 'Email field must be completed',
    });
    expect(submitSpy).toNotHaveBeenCalled();
  });

  it('submits the form data when the form is submitted', () => {
    const submitSpy = createSpy();
    const form = mount(<ForgotPasswordForm handleSubmit={submitSpy} />);
    const emailField = form.find({ name: 'email' });
    const submitBtn = form.find('button');

    fillInFormInput(emailField, email);
    submitBtn.simulate('submit');

    expect(submitSpy).toHaveBeenCalledWith({ email });
  });

  it('does not submit the form if the email is not valid', () => {
    const submitSpy = createSpy();
    const form = mount(<ForgotPasswordForm handleSubmit={submitSpy} />);
    const emailField = form.find({ name: 'email' });
    const submitBtn = form.find('button');

    fillInFormInput(emailField, 'invalid-email');
    submitBtn.simulate('submit');

    expect(submitSpy).toNotHaveBeenCalled();
    expect(form.state().errors).toInclude({
      email: 'invalid-email is not a valid email',
    });
  });
});

