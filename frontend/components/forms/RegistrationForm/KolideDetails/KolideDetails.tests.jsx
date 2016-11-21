import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import KolideDetails from 'components/forms/RegistrationForm/KolideDetails';
import { fillInFormInput } from 'test/helpers';

describe('KolideDetails - form', () => {
  afterEach(restoreSpies);

  describe('kolide web address input', () => {
    it('renders an input field', () => {
      const form = mount(<KolideDetails handleSubmit={noop} />);
      const kolideWebAddressField = form.find({ name: 'kolide_server_url' });

      expect(kolideWebAddressField.length).toEqual(1);
    });

    it('updates state when the field changes', () => {
      const form = mount(<KolideDetails handleSubmit={noop} />);
      const kolideWebAddressField = form.find({ name: 'kolide_server_url' }).find('input');

      fillInFormInput(kolideWebAddressField, 'https://gnar.kolide.co');

      expect(form.state().formData).toInclude({ kolide_server_url: 'https://gnar.kolide.co' });
    });
  });

  describe('submitting the form', () => {
    it('validates the presence of the kolide web address field', () => {
      const handleSubmitSpy = createSpy();
      const form = mount(<KolideDetails handleSubmit={handleSubmitSpy} />);
      const submitBtn = form.find('Button');

      submitBtn.simulate('click');

      expect(handleSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({ kolide_server_url: 'Kolide web address must be completed' });
    });

    it('validates the kolide web address field starts with https://', () => {
      const handleSubmitSpy = createSpy();
      const form = mount(<KolideDetails handleSubmit={handleSubmitSpy} />);
      const kolideWebAddressField = form.find({ name: 'kolide_server_url' }).find('input');
      const submitBtn = form.find('Button');

      fillInFormInput(kolideWebAddressField, 'http://gnar.kolide.co');
      submitBtn.simulate('click');

      expect(handleSubmitSpy).toNotHaveBeenCalled();
      expect(form.state().errors).toInclude({
        kolide_server_url: 'Kolide web address must start with https://',
      });
    });

    it('submits the form when valid', () => {
      const handleSubmitSpy = createSpy();
      const form = mount(<KolideDetails handleSubmit={handleSubmitSpy} />);
      const kolideWebAddressField = form.find({ name: 'kolide_server_url' }).find('input');
      const submitBtn = form.find('Button');

      fillInFormInput(kolideWebAddressField, 'https://gnar.kolide.co');
      submitBtn.simulate('click');

      expect(handleSubmitSpy).toHaveBeenCalled();
    });
  });
});

