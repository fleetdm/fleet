import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { fillInFormInput } from '../../../test/helpers';
import SaveQueryFormModal from './SaveQueryFormModal';

describe('SaveQueryFormModal - component', () => {
  afterEach(restoreSpies);

  it('renders the SaveQueryForm in a Modal', () => {
    const component = mount(<SaveQueryFormModal />);
    const modal = component.find('Modal');

    expect(modal.length).toEqual(1);
    expect(modal.find('SaveQueryForm').length).toEqual(1);
  });

  it('calls onCancel prop when exiting the modal', () => {
    const onCancelSpy = createSpy();
    const component = mount(<SaveQueryFormModal onCancel={onCancelSpy} />);
    const modal = component.find('Modal');

    modal.find('.modal__ex').simulate('click');

    expect(onCancelSpy).toHaveBeenCalled();
  });

  it('calls then onCancel prop when cancelling the form', () => {
    const onCancelSpy = createSpy();
    const component = mount(<SaveQueryFormModal onCancel={onCancelSpy} />);
    const form = component.find('SaveQueryForm');
    const cancelBtn = form.find('.save-query-form__btn--cancel');

    cancelBtn.simulate('click');

    expect(onCancelSpy).toHaveBeenCalled();
  });

  it('calls the onSubmit prop when submitting a valid form', () => {
    const onSubmitSpy = createSpy();
    const component = mount(<SaveQueryFormModal onSubmit={onSubmitSpy} />);
    const form = component.find('SaveQueryForm');
    const nameInputField = form.find('.save-query-form__input--name');
    const descriptionInputField = form.find('.save-query-form__input--description');

    fillInFormInput(nameInputField, 'my query');
    fillInFormInput(descriptionInputField, 'my description');
    form.simulate('submit');

    expect(onSubmitSpy).toHaveBeenCalledWith({
      name: 'my query',
      description: 'my description',
    });
  });
});
