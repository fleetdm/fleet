import React from 'react';
import { mount } from 'enzyme';

import RegistrationForm from 'components/forms/RegistrationForm';

describe('RegistrationForm - component', () => {
  it('renders AdminDetails and header on the first page', () => {
    const form = mount(<RegistrationForm page={1} />);

    expect(form.find('AdminDetails').length).toEqual(1);
    expect(form.text()).toContain('SET USERNAME & PASSWORD');
  });

  it('renders OrgDetails on the second page', () => {
    const form = mount(<RegistrationForm page={2} />);

    expect(form.find('OrgDetails').length).toEqual(1);
    expect(form.text()).toContain('SET ORGANIZATION DETAILS');
  });

  it('renders KolideDetails on the third page', () => {
    const form = mount(<RegistrationForm page={3} />);

    expect(form.find('KolideDetails').length).toEqual(1);
    expect(form.text()).toContain('SET KOLIDE WEB ADDRESS');
  });

  it('renders ConfirmationPage on the fourth page', () => {
    const form = mount(<RegistrationForm page={4} />);

    expect(form.find('ConfirmationPage').length).toEqual(1);
    expect(form.text()).toContain('SUCCESS');
  });
});

