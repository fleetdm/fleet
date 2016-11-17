import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { fillInFormInput } from 'test/helpers';
import QueryForm from './index';

const query = {
  id: 1,
  name: 'All users',
  description: 'Query to get all users',
  query: 'SELECT * FROM users',
};
const queryText = 'SELECT * FROM users';

describe('QueryForm - component', () => {
  afterEach(restoreSpies);

  it('renders InputFields for the query name and description', () => {
    const form = mount(<QueryForm query={query} queryText={queryText} />);
    const inputFields = form.find('InputField');

    expect(inputFields.length).toEqual(2);
    expect(inputFields.find({ name: 'name' }).length).toEqual(1);
    expect(inputFields.find({ name: 'description' }).length).toEqual(1);
  });

  it('updates state on input field change', () => {
    const form = mount(<QueryForm query={query} queryText={queryText} />);
    const inputFields = form.find('InputField');
    const nameInput = inputFields.find({ name: 'name' });
    const descriptionInput = inputFields.find({ name: 'description' });
    fillInFormInput(nameInput, 'new name');
    fillInFormInput(descriptionInput, 'new description');

    expect(form.state()).toInclude({
      formData: {
        description: 'new description',
        name: 'new name',
        query: queryText,
      },
    });
  });

  it('validates the query name before saving changes', () => {
    const onSaveChangesSpy = createSpy();
    const form = mount(<QueryForm query={query} queryText={queryText} onUpdate={onSaveChangesSpy} />);
    const inputFields = form.find('InputField');
    const nameInput = inputFields.find({ name: 'name' });

    fillInFormInput(nameInput, '');

    const saveChangesBtn = form.find('.query-form__save-changes-btn');

    saveChangesBtn.simulate('click');

    expect(onSaveChangesSpy).toNotHaveBeenCalled();
    expect(form.state()).toInclude({
      errors: {
        name: 'Query title must be present',
        description: null,
      },
    });
  });

  it('calls the onSaveChanges prop when the form is valid', () => {
    const onSaveChangesSpy = createSpy();
    const form = mount(<QueryForm query={query} queryText={queryText} onUpdate={onSaveChangesSpy} />);
    const inputFields = form.find('InputField');
    const nameInput = inputFields.find({ name: 'name' });

    fillInFormInput(nameInput, 'New query name');

    const saveChangesBtn = form.find('.query-form__save-changes-btn');

    saveChangesBtn.simulate('click');

    expect(onSaveChangesSpy).toHaveBeenCalledWith({
      description: query.description,
      name: 'New query name',
      query: queryText,
    });
  });

  it('enables the Save Changes button when the name input changes', () => {
    const form = mount(<QueryForm query={query} queryText={queryText} />);
    const inputFields = form.find('InputField');
    const nameInput = inputFields.find({ name: 'name' });
    const saveChangesBtn = form.find('.query-form__save-changes-btn');

    expect(saveChangesBtn.props()).toInclude({
      disabled: true,
    });

    fillInFormInput(nameInput, 'New query name');

    expect(saveChangesBtn.props()).toNotInclude({
      disabled: true,
    });
  });

  it('enables the Save Changes button when the description input changes', () => {
    const form = mount(<QueryForm query={query} queryText={queryText} />);
    const inputFields = form.find('InputField');
    const descriptionInput = inputFields.find({ name: 'description' });
    const saveChangesBtn = form.find('.query-form__save-changes-btn');

    expect(saveChangesBtn.props()).toInclude({
      disabled: true,
    });

    fillInFormInput(descriptionInput, 'New query description');

    expect(saveChangesBtn.props()).toNotInclude({
      disabled: true,
    });
  });

  it('calls the onSaveAsNew prop when "Save As New" is clicked and the form is valid', () => {
    const onSaveAsNewSpy = createSpy();
    const form = mount(<QueryForm query={query} queryText={queryText} onSave={onSaveAsNewSpy} />);
    const inputFields = form.find('InputField');
    const nameInput = inputFields.find({ name: 'name' });
    const saveAsNewBtn = form.find('.query-form__save-as-new-btn');

    fillInFormInput(nameInput, 'New query name');

    saveAsNewBtn.simulate('click');

    expect(onSaveAsNewSpy).toHaveBeenCalledWith({
      description: query.description,
      name: 'New query name',
      query: queryText,
    });
  });

  it('does not call the onSaveAsNew prop when "Save As New" is clicked and the form is not valid', () => {
    const onSaveAsNewSpy = createSpy();
    const form = mount(<QueryForm query={query} queryText={queryText} onSave={onSaveAsNewSpy} />);
    const inputFields = form.find('InputField');
    const nameInput = inputFields.find({ name: 'name' });
    const saveAsNewBtn = form.find('.query-form__save-as-new-btn');

    fillInFormInput(nameInput, '');

    saveAsNewBtn.simulate('click');

    expect(onSaveAsNewSpy).toNotHaveBeenCalled();
    expect(form.state()).toInclude({
      errors: {
        name: 'Query title must be present',
        description: null,
      },
    });
  });

  it('calls the onRunQuery prop when "Run Query" is clicked and the form is valid', () => {
    const onRunQuerySpy = createSpy();
    const form = mount(<QueryForm query={query} queryText={queryText} onRunQuery={onRunQuerySpy} />);
    const runQueryBtn = form.find('.query-form__run-query-btn');

    runQueryBtn.simulate('click');

    expect(onRunQuerySpy).toHaveBeenCalled();
  });
});
