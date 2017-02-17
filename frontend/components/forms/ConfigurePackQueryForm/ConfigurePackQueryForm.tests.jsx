import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConfigurePackQueryForm from 'components/forms/ConfigurePackQueryForm';
import { itBehavesLikeAFormDropdownElement, itBehavesLikeAFormInputElement } from 'test/helpers';
import { scheduledQueryStub } from 'test/stubs';

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
      itBehavesLikeAFormInputElement(form, 'shard');
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
      itBehavesLikeAFormInputElement(form, 'shard', 'InputField', 12);

      form.find('form').simulate('submit');

      expect(spy).toHaveBeenCalledWith({
        interval: 123,
        logging_type: 'differential',
        platform: '',
        query_id: 1,
        version: '',
        shard: 12,
      });
    });
  });

  describe('cancelling the form', () => {
    const CancelButton = form => form.find('.configure-pack-query-form__cancel-btn');

    it('displays a cancel Button when updating a scheduled query', () => {
      const NewScheduledQueryForm = mount(
        <ConfigurePackQueryForm
          formData={{ query_id: 1 }}
          handleSubmit={noop}
          onCancel={noop}
        />
      );
      const UpdateScheduledQueryForm = mount(
        <ConfigurePackQueryForm
          formData={scheduledQueryStub}
          handleSubmit={noop}
          onCancel={noop}
        />
      );

      expect(CancelButton(NewScheduledQueryForm).length).toEqual(0);
      expect(CancelButton(UpdateScheduledQueryForm).length).toEqual(1);
    });

    it('calls the onCancel prop when the cancel Button is clicked', () => {
      const spy = createSpy();
      const UpdateScheduledQueryForm = mount(
        <ConfigurePackQueryForm
          formData={scheduledQueryStub}
          handleSubmit={noop}
          onCancel={spy}
        />
      );

      CancelButton(UpdateScheduledQueryForm).simulate('click');

      expect(spy).toHaveBeenCalledWith(scheduledQueryStub);
    });
  });
});
