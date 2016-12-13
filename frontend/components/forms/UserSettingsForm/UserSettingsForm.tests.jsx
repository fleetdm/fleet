import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import UserSettingsForm from 'components/forms/UserSettingsForm';
import helpers from 'test/helpers';

const { fillInFormInput, itBehavesLikeAFormInputElement } = helpers;

describe('UserSettingsForm - component', () => {
  afterEach(restoreSpies);

  it('has the correct fields', () => {
    const form = mount(<UserSettingsForm handleSubmit={noop} />);

    itBehavesLikeAFormInputElement(form, 'email');
    itBehavesLikeAFormInputElement(form, 'name');
    itBehavesLikeAFormInputElement(form, 'username');
  });

  it('calls the handleSubmit props with form data', () => {
    const handleSubmitSpy = createSpy();
    const form = mount(<UserSettingsForm handleSubmit={handleSubmitSpy} />);
    const expectedFormData = { email: 'email@example.com', name: 'Jim Example', username: 'jimmyexamples' };
    const emailInput = form.find({ name: 'email' }).find('input');
    const nameInput = form.find({ name: 'name' }).find('input');
    const usernameInput = form.find({ name: 'username' }).find('input');

    fillInFormInput(emailInput, expectedFormData.email);
    fillInFormInput(nameInput, expectedFormData.name);
    fillInFormInput(usernameInput, expectedFormData.username);

    form.simulate('submit');

    expect(handleSubmitSpy).toHaveBeenCalledWith(expectedFormData);
  });

  it('initializes the form with the users data', () => {
    const user = { email: 'email@example.com', name: 'Jim Example', username: 'jimmyexamples' };
    const form = mount(<UserSettingsForm formData={user} handleSubmit={noop} />);

    expect(form.state().formData).toEqual(user);
  });
});
