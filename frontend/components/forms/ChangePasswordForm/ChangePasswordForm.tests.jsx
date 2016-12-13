import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ChangePasswordForm from 'components/forms/ChangePasswordForm';
import helpers from 'test/helpers';

const { fillInFormInput, itBehavesLikeAFormInputElement } = helpers;

describe('ChangePasswordForm - component', () => {
  afterEach(restoreSpies);

  it('has the correct fields', () => {
    const form = mount(<ChangePasswordForm handleSubmit={noop} onCancel={noop} />);

    itBehavesLikeAFormInputElement(form, 'password');
    itBehavesLikeAFormInputElement(form, 'new_password');
    itBehavesLikeAFormInputElement(form, 'new_password_confirmation');
  });

  it('calls the handleSubmit props with form data', () => {
    const handleSubmitSpy = createSpy();
    const form = mount(<ChangePasswordForm handleSubmit={handleSubmitSpy} onCancel={noop} />);
    const expectedFormData = { password: 'password', new_password: 'new_password', new_password_confirmation: 'new_password' };
    const passwordInput = form.find({ name: 'password' }).find('input');
    const newPasswordInput = form.find({ name: 'new_password' }).find('input');
    const newPasswordConfirmationInput = form.find({ name: 'new_password_confirmation' }).find('input');

    fillInFormInput(passwordInput, expectedFormData.password);
    fillInFormInput(newPasswordInput, expectedFormData.new_password);
    fillInFormInput(newPasswordConfirmationInput, expectedFormData.new_password_confirmation);

    form.simulate('submit');

    expect(handleSubmitSpy).toHaveBeenCalledWith(expectedFormData);
  });

  it('calls the onCancel prop when CANCEL is clicked', () => {
    const onCancelSpy = createSpy();
    const form = mount(<ChangePasswordForm handleSubmit={noop} onCancel={onCancelSpy} />);
    const cancelBtn = form.find('Button').findWhere(n => n.prop('text') === 'CANCEL').find('button');

    cancelBtn.simulate('click');

    expect(onCancelSpy).toHaveBeenCalled();
  });
});

