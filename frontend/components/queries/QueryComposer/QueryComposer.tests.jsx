import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { createAceSpy, fillInFormInput } from 'test/helpers';
import QueryComposer from './index';

describe('QueryComposer - component', () => {
  beforeEach(() => {
    createAceSpy();
  });
  afterEach(restoreSpies);

  it('does not render the SaveQueryForm by default', () => {
    const component = mount(
      <QueryComposer
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        selectedTargets={[]}
        queryText="Hello world"
      />
    );

    expect(component.find('SaveQueryForm').length).toEqual(0);
  });

  it('renders the QueryForm when the query prop is present', () => {
    const query = {
      id: 1,
      query: 'SELECT * FROM users',
      name: 'Get all users',
      description: 'This gets all of the users',
    };
    const component = mount(
      <QueryComposer
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        query={query}
        selectedTargets={[]}
        queryText="Hello world"
      />
    );

    const form = component.find('QueryForm');

    expect(form.length).toEqual(1);
    expect(form.find('InputField').length).toEqual(2);
  });

  it('calls onSaveQueryFormSubmit with appropriate data when "Save as New" is clicked', () => {
    const onSaveQueryFormSubmitSpy = createSpy();
    const queryText = 'SELECT * FROM users';
    const selectedTargets = [{ name: 'my target' }];
    const component = mount(
      <QueryComposer
        onSave={onSaveQueryFormSubmitSpy}
        selectedTargets={selectedTargets}
        queryText={queryText}
      />
    );

    const form = component.find('QueryForm');

    fillInFormInput(form.find({ name: 'name' }), 'My query name');
    fillInFormInput(form.find({ name: 'description' }), 'My query description');
    form.find('.query-form__save-as-new-btn').simulate('click');

    expect(onSaveQueryFormSubmitSpy).toHaveBeenCalledWith({
      description: 'My query description',
      name: 'My query name',
      query: queryText,
    });
  });

  it('calls onRunQuery when "Run Query" is clicked', () => {
    const onRunQuerySpy = createSpy();
    const query = 'SELECT * FROM users';
    const selectedTargets = [{ name: 'my target' }];
    const component = mount(
      <QueryComposer
        onRunQuery={onRunQuerySpy}
        selectedTargets={selectedTargets}
        queryText={query}
      />
    );
    component.find('.query-form__run-query-btn').simulate('click');

    expect(onRunQuerySpy).toHaveBeenCalled();
  });

  it('calls onSaveChanges when "Save Changes" is clicked', () => {
    const onSaveChangesSpy = createSpy();
    const query = {
      name: 'my query',
      description: 'my description',
      query: 'SELECT * FROM users',
    };
    const component = mount(
      <QueryComposer
        onUpdate={onSaveChangesSpy}
        query={query}
        queryText={query.query}
      />
    );
    const form = component.find('QueryForm');

    fillInFormInput(form.find({ name: 'name' }), 'My new query name');
    form.find('.query-form__save-changes-btn').simulate('click');

    expect(onSaveChangesSpy).toHaveBeenCalledWith({
      description: query.description,
      name: 'My new query name',
      query: query.query,
    });
  });
});
