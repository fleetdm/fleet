import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import AdminDetails from 'components/forms/RegistrationForm/AdminDetails';
import { fillInFormInput, itBehavesLikeAFormInputElement } from 'test/helpers';

describe('AdminDetails - form', () => {
  afterEach(restoreSpies);

  let form = mount(<AdminDetails handleSubmit={noop} />);

  describe('username input', () => {
    it('renders an input field', () => {
      itBehavesLikeAFormInputElement(form, 'username');
    });
  });

  describe('password input', () => {
    it('renders an input field', () => {
      itBehavesLikeAFormInputElement(form, 'password');
    });
  });

  describe('password confirmation input', () => {
    it('renders an input field', () => {
      itBehavesLikeAFormInputElement(form, 'password_confirmation');
    });
  });

  describe('email input', () => {
    it('renders an input field', () => {
      itBehavesLikeAFormInputElement(form, 'email');
    });
  });

  describe('submitting the form', () => {
    it('validates the email field', () => {
      const onSubmitSpy = createSpy();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const submitBtn = form.find('Button');


      submitBtn.simulate('click');

      expect(onSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({
        email: 'Email must be present',
        password: 'Password must be present',
        password_confirmation: 'Password confirmation must be present',
        username: 'Username must be present',
      });
    });

    it('validates the email field', () => {
      const onSubmitSpy = createSpy();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const emailField = form.find({ name: 'email' }).find('input');
      const submitBtn = form.find('Button');

      fillInFormInput(emailField, 'invalid-email');
      submitBtn.simulate('click');

      expect(onSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({ email: 'Email must be a valid email' });
    });

    it('validates the password fields match', () => {
      const onSubmitSpy = createSpy();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const passwordConfirmationField = form.find({ name: 'password_confirmation' }).find('input');
      const passwordField = form.find({ name: 'password' }).find('input');
      const submitBtn = form.find('Button');

      fillInFormInput(passwordField, 'p@ssw0rd');
      fillInFormInput(passwordConfirmationField, 'password123');
      submitBtn.simulate('click');

      expect(onSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({
        password_confirmation: 'Password confirmation does not match password',
      });
    });

    it('submits the form when valid', () => {
      const onSubmitSpy = createSpy();
      form = mount(<AdminDetails handleSubmit={onSubmitSpy} />);
      const emailField = form.find({ name: 'email' }).find('input');
      const passwordConfirmationField = form.find({ name: 'password_confirmation' }).find('input');
      const passwordField = form.find({ name: 'password' }).find('input');
      const usernameField = form.find({ name: 'username' }).find('input');
      const submitBtn = form.find('Button');

      fillInFormInput(emailField, 'hi@gnar.dog');
      fillInFormInput(passwordField, 'p@ssw0rd');
      fillInFormInput(passwordConfirmationField, 'p@ssw0rd');
      fillInFormInput(usernameField, 'gnardog');
      submitBtn.simulate('click');

      expect(onSubmitSpy).toHaveBeenCalled();
    });
  });
});
