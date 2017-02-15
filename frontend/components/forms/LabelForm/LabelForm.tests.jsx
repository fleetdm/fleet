import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import LabelForm from 'components/forms/LabelForm';
import {
  fillInFormInput,
  itBehavesLikeAFormDropdownElement,
  itBehavesLikeAFormInputElement,
} from 'test/helpers';

describe('LabelForm - form', () => {
  afterEach(restoreSpies);

  const defaultProps = {
    handleSubmit: noop,
    isEdit: false,
    onCancel: noop,
  };

  describe('inputs', () => {
    const form = mount(<LabelForm {...defaultProps} />);

    describe('name input', () => {
      it('renders an input field', () => {
        itBehavesLikeAFormInputElement(form, 'name');
      });
    });

    describe('description input', () => {
      it('renders an input field', () => {
        itBehavesLikeAFormInputElement(form, 'description', 'textarea');
      });
    });

    describe('platform input', () => {
      it('renders an input field', () => {
        itBehavesLikeAFormDropdownElement(form, 'platform');
      });
    });
  });

  describe('submitting the form', () => {
    it('validates the name field', () => {
      const spy = createSpy();
      const props = { ...defaultProps, handleSubmit: spy };
      const form = mount(<LabelForm {...props} />);
      const htmlForm = form.find('form');


      htmlForm.simulate('submit');

      expect(spy).toNotHaveBeenCalled();
      expect(form.state('errors')).toInclude({
        name: 'Label title must be present',
      });
    });

    it('submits the form when valid', () => {
      const spy = createSpy();
      const props = { ...defaultProps, handleSubmit: spy, formData: { query: 'select * from users' } };
      const form = mount(<LabelForm {...props} />);
      const nameField = form.find({ name: 'name' }).find('input');
      const descriptionField = form.find({ name: 'description' }).find('textarea');
      const htmlForm = form.find('form');

      fillInFormInput(nameField, 'My new label');
      fillInFormInput(descriptionField, 'This is my new label');
      itBehavesLikeAFormDropdownElement(form, 'platform');
      htmlForm.simulate('submit');

      expect(spy).toHaveBeenCalledWith({
        name: 'My new label',
        description: 'This is my new label',
        platform: '',
        query: 'select * from users',
      });
    });
  });
});

