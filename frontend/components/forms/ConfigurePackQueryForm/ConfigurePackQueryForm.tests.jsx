import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConfigurePackQueryForm from 'components/forms/ConfigurePackQueryForm';
import { itBehavesLikeAFormDropdownElement, itBehavesLikeAFormInputElement } from 'test/helpers';

describe('ConfigurePackQueryForm - component', () => {
  afterEach(restoreSpies);

  describe('form fields', () => {
    const form = mount(
      <ConfigurePackQueryForm
        handleSubmit={noop}
      />
    );

    it('updates form state', () => {
      itBehavesLikeAFormInputElement(form, 'interval');
      itBehavesLikeAFormDropdownElement(form, 'logging_type');
      itBehavesLikeAFormDropdownElement(form, 'platform');
      itBehavesLikeAFormDropdownElement(form, 'version');
    });
  });

  describe('submitting the form', () => {
    const spy = createSpy();
    const form = mount(
      <ConfigurePackQueryForm
        handleSubmit={spy}
        formData={{ query_id: 1 }}
      />
    );

    it('submits the form with the form data', () => {
      itBehavesLikeAFormInputElement(form, 'interval', 'InputField', 123);
      itBehavesLikeAFormDropdownElement(form, 'logging_type');
      itBehavesLikeAFormDropdownElement(form, 'platform');
      itBehavesLikeAFormDropdownElement(form, 'version');

      form.find('form').simulate('submit');

      expect(spy).toHaveBeenCalledWith({
        interval: 123,
        logging_type: 'differential',
        platform: '',
        query_id: 1,
        version: '',
      });
    });
  });
});
