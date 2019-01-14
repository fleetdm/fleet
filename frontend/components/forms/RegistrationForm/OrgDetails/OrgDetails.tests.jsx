import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import OrgDetails from 'components/forms/RegistrationForm/OrgDetails';
import { fillInFormInput } from 'test/helpers';

describe('OrgDetails - form', () => {
  afterEach(restoreSpies);

  describe('organization name input', () => {
    it('renders an input field', () => {
      const form = mount(<OrgDetails handleSubmit={noop} />);
      const orgNameField = form.find({ name: 'org_name' });

      expect(orgNameField.length).toBeGreaterThan(0);
    });

    it('updates state when the field changes', () => {
      const form = mount(<OrgDetails handleSubmit={noop} />);
      const orgNameField = form.find({ name: 'org_name' }).find('input');

      fillInFormInput(orgNameField, 'The Gnar Co');

      expect(form.state().formData).toInclude({ org_name: 'The Gnar Co' });
    });
  });

  describe('organization logo URL input', () => {
    it('renders an input field', () => {
      const form = mount(<OrgDetails handleSubmit={noop} />);
      const orgLogoField = form.find({ name: 'org_logo_url' });

      expect(orgLogoField.length).toBeGreaterThan(0);
    });

    it('updates state when the field changes', () => {
      const form = mount(<OrgDetails handleSubmit={noop} />);
      const orgLogoField = form.find({ name: 'org_logo_url' }).find('input');

      fillInFormInput(orgLogoField, 'http://www.thegnar.co/logo.png');

      expect(form.state().formData).toInclude({ org_logo_url: 'http://www.thegnar.co/logo.png' });
    });
  });

  describe('submitting the form', () => {
    it('validates presence of org_name field', () => {
      const handleSubmitSpy = createSpy();
      const form = mount(<OrgDetails handleSubmit={handleSubmitSpy} />);
      const htmlForm = form.find('form');

      htmlForm.simulate('submit');

      expect(handleSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({
        org_name: 'Organization name must be present',
      });
    });

    it('validates the logo url field starts with https://', () => {
      const handleSubmitSpy = createSpy();
      const form = mount(<OrgDetails handleSubmit={handleSubmitSpy} />);
      const orgLogoField = form.find({ name: 'org_logo_url' }).find('input');
      const htmlForm = form.find('form');

      fillInFormInput(orgLogoField, 'http://www.thegnar.co/logo.png');
      htmlForm.simulate('submit');

      expect(handleSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({
        org_logo_url: 'Organization logo URL must start with https://',
      });
    });

    it('submits the form when valid', () => {
      const handleSubmitSpy = createSpy();
      const form = mount(<OrgDetails handleSubmit={handleSubmitSpy} />);
      const orgLogoField = form.find({ name: 'org_logo_url' }).find('input');
      const orgNameField = form.find({ name: 'org_name' }).find('input');
      const htmlForm = form.find('form');

      fillInFormInput(orgLogoField, 'https://www.thegnar.co/logo.png');
      fillInFormInput(orgNameField, 'The Gnar Co');

      htmlForm.simulate('submit');

      expect(handleSubmitSpy).toHaveBeenCalled();
    });
  });
});
