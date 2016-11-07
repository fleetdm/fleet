import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import EditUserForm from './EditUserForm';
import { fillInFormInput } from '../../../../test/helpers';

describe('EditUserForm - form', () => {
  afterEach(restoreSpies);

  const user = {
    email: 'hi@gnar.dog',
    name: 'Gnar Dog',
    position: 'Head of Everything',
    username: 'gnardog',
  };

  it('sends the users changed attributes when the form is submitted', () => {
    const email = 'newEmail@gnar.dog';
    const onSubmit = createSpy();
    const form = mount(<EditUserForm user={user} onSubmit={onSubmit} />);
    const emailInput = form.find({ name: 'email' });

    fillInFormInput(emailInput, email);
    form.simulate('submit');

    expect(onSubmit).toHaveBeenCalledWith({
      ...user,
      email,
    });
  });
});
