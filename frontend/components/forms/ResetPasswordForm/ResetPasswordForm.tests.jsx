import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import ResetPasswordForm from './ResetPasswordForm';
import { fillInFormInput } from '../../../test/helpers';

describe('ResetPasswordForm - component', () => {
  const newPassword = 'my new password';

  afterEach(restoreSpies);

  it('updates component state when the new_password field is changed', () => {
    const form = mount(<ResetPasswordForm onSubmit={noop} />);

    const newPasswordField = form.find({ name: 'new_password' });
    fillInFormInput(newPasswordField, newPassword);

    const { formData } = form.state();
    expect(formData).toContain({ new_password: newPassword });
  });

  it('updates component state when the new_password_confirmation field is changed', () => {
    const form = mount(<ResetPasswordForm onSubmit={noop} />);

    const newPasswordField = form.find({ name: 'new_password_confirmation' });
    fillInFormInput(newPasswordField, newPassword);

    const { formData } = form.state();
    expect(formData).toContain({ new_password_confirmation: newPassword });
  });

  it('it does not submit the form when the form fields have not been filled out', () => {
    const submitSpy = createSpy();
    const form = mount(<ResetPasswordForm onSubmit={submitSpy} />);
    const submitBtn = form.find('button');

    submitBtn.simulate('submit');

    const { errors } = form.state();
    expect(errors.new_password).toEqual('New Password field must be completed');
    expect(submitSpy).toNotHaveBeenCalled();
  });

  it('it does not submit the form when only the new password field has been filled out', () => {
    const submitSpy = createSpy();
    const form = mount(<ResetPasswordForm onSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: 'new_password' });
    fillInFormInput(newPasswordField, newPassword);
    const submitBtn = form.find('button');

    submitBtn.simulate('submit');

    const { errors } = form.state();
    expect(errors.new_password_confirmation).toEqual('New Password Confirmation field must be completed');
    expect(submitSpy).toNotHaveBeenCalled();
  });

  it('submits the form data when the form is submitted', () => {
    const submitSpy = createSpy();
    const form = mount(<ResetPasswordForm onSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: 'new_password' });
    const newPasswordConfirmationField = form.find({ name: 'new_password_confirmation' });
    const submitBtn = form.find('button');

    fillInFormInput(newPasswordField, newPassword);
    fillInFormInput(newPasswordConfirmationField, newPassword);
    submitBtn.simulate('submit');

    expect(submitSpy).toHaveBeenCalledWith({
      new_password: newPassword,
      new_password_confirmation: newPassword,
    });
  });

  it('does not submit the form if the new password confirmation does not match', () => {
    const submitSpy = createSpy();
    const form = mount(<ResetPasswordForm onSubmit={submitSpy} />);
    const newPasswordField = form.find({ name: 'new_password' });
    const newPasswordConfirmationField = form.find({ name: 'new_password_confirmation' });
    const submitBtn = form.find('button');

    fillInFormInput(newPasswordField, newPassword);
    fillInFormInput(newPasswordConfirmationField, 'not my new password');
    submitBtn.simulate('submit');

    expect(submitSpy).toNotHaveBeenCalled();
    expect(form.state().errors).toInclude({
      new_password_confirmation: 'Passwords Do Not Match',
    });
  });
});
