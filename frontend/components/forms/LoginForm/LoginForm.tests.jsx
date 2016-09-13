import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import LoginForm from './LoginForm';
import { fillInFormInput } from '../../../test/helpers';

describe('LoginForm - component', () => {
  afterEach(restoreSpies);

  it('renders 2 InputFieldWithIcon components', () => {
    const form = mount(<LoginForm onSubmit={noop} />);

    expect(form.find('InputFieldWithIcon').length).toEqual(2);
  });

  it('updates component state when the username field is changed', () => {
    const form = mount(<LoginForm onSubmit={noop} />);
    const username = 'hi@thegnar.co';

    const usernameField = form.find({ name: 'username' });
    fillInFormInput(usernameField, username);

    const { formData } = form.state();
    expect(formData).toContain({ username });
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

    expect(submitBtn.prop('style')).toInclude({
      cursor: 'not-allowed',
    });
    expect(submitSpy).toNotHaveBeenCalled();
  });

  it('submits the form data when form is submitted', () => {
    const submitSpy = createSpy();
    const form = mount(<LoginForm onSubmit={submitSpy} />);
    const usernameField = form.find({ name: 'username' });
    const passwordField = form.find({ name: 'password' });
    const submitBtn = form.find('button');

    fillInFormInput(usernameField, 'my@email.com');
    fillInFormInput(passwordField, 'p@ssw0rd');
    submitBtn.simulate('submit');

    expect(submitSpy).toHaveBeenCalledWith({
      username: 'my@email.com',
      password: 'p@ssw0rd',
    });
  });
});
