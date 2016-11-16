import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConfirmationPage from 'components/forms/RegistrationForm/ConfirmationPage';

describe('ConfirmationPage - form', () => {
  afterEach(restoreSpies);

  const formData = {
    full_name: 'Jason Meller',
    username: 'jmeller',
    email: 'jason@kolide.co',
    org_name: 'Kolide',
    kolide_web_address: 'http://kolide.kolide.co',
  };

  it('renders the user information', () => {
    const form = mount(
      <ConfirmationPage
        formData={formData}
        handleSubmit={noop}
      />
    );

    expect(form.text()).toInclude(formData.full_name);
    expect(form.text()).toInclude(formData.username);
    expect(form.text()).toInclude(formData.email);
    expect(form.text()).toInclude(formData.org_name);
    expect(form.text()).toInclude(formData.kolide_web_address);
  });

  it('submits the form', () => {
    const handleSubmitSpy = createSpy();
    const form = mount(
      <ConfirmationPage
        formData={formData}
        handleSubmit={handleSubmitSpy}
      />
    );
    const submitBtn = form.find('Button');

    submitBtn.simulate('click');

    expect(handleSubmitSpy).toHaveBeenCalled();
  });
});

