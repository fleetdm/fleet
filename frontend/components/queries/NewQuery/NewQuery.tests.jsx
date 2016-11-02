import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { createAceSpy, fillInFormInput } from 'test/helpers';
import NewQuery from './index';

describe('NewQuery - component', () => {
  beforeEach(() => {
    createAceSpy();
  });
  afterEach(restoreSpies);

  it('renders the ThemeDropdown', () => {
    const component = mount(
      <NewQuery
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        textEditorText="Hello world"
      />
    );

    expect(component.find('ThemeDropdown').length).toEqual(1);
  });

  it('does not render the SaveQueryForm by default', () => {
    const component = mount(
      <NewQuery
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        textEditorText="Hello world"
      />
    );

    expect(component.find('SaveQueryForm').length).toEqual(0);
  });

  it('renders the SaveQueryFormModal when "Save Query" is clicked', () => {
    const component = mount(
      <NewQuery
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        textEditorText="Hello world"
      />
    );

    component.find('.new-query__save-query-btn').simulate('click');

    expect(component.find('SaveQueryForm').length).toEqual(1);
  });

  it('renders the Run Query button as disabled without selected targets', () => {
    const component = mount(
      <NewQuery
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        textEditorText="Hello world"
      />
    );

    const runQueryBtn = component.find('.new-query__run-query-btn');

    expect(runQueryBtn.props()).toInclude({
      disabled: true,
    });
  });

  it('hides the SaveQueryFormModal after the form is submitted', () => {
    const component = mount(
      <NewQuery onNewQueryFormSubmit={noop} textEditorText="SELECT * FROM users" />
    );

    component.find('.new-query__save-query-btn').simulate('click');

    const form = component.find('SaveQueryForm');

    fillInFormInput(form.find({ name: 'name' }), 'My query name');
    form.simulate('submit');

    expect(component.find('SaveQueryForm').length).toEqual(0);
  });

  it('calls onNewQueryFormSubmit with appropriate data from SaveQueryFormModal', () => {
    const onNewQueryFormSubmitSpy = createSpy();
    const query = 'SELECT * FROM users';
    const selectedTargets = [{ name: 'my target' }];
    const component = mount(
      <NewQuery
        onNewQueryFormSubmit={onNewQueryFormSubmitSpy}
        textEditorText={query}
      />
    );

    component.setState({ selectedTargets });
    component.find('.new-query__save-query-btn').simulate('click');

    const form = component.find('SaveQueryForm');

    fillInFormInput(form.find({ name: 'name' }), 'My query name');
    fillInFormInput(form.find({ name: 'description' }), 'My query description');
    form.simulate('submit');

    expect(onNewQueryFormSubmitSpy).toHaveBeenCalledWith({
      description: 'My query description',
      name: 'My query name',
      query,
      selectedTargets,
    });
  });

  it('calls onNewQueryFormSubmit when "Run Query" is clicked', () => {
    const onNewQueryFormSubmitSpy = createSpy();
    const query = 'SELECT * FROM users';
    const selectedTargets = [{ name: 'my target' }];
    const component = mount(
      <NewQuery
        onNewQueryFormSubmit={onNewQueryFormSubmitSpy}
        textEditorText={query}
      />
    );
    component.setState({ selectedTargets });
    component.find('.new-query__run-query-btn').simulate('click');

    expect(onNewQueryFormSubmitSpy).toHaveBeenCalledWith({ query, selectedTargets });
  });

  it('calls onTargetSelectInputChange when changing the select target input text', () => {
    const onTargetSelectInputChangeSpy = createSpy();
    const component = mount(
      <NewQuery onTargetSelectInputChange={onTargetSelectInputChangeSpy} />
    );
    const selectTargetsInput = component.find('.Select-input input');

    fillInFormInput(selectTargetsInput, 'my target');

    expect(onTargetSelectInputChangeSpy).toHaveBeenCalledWith('my target');
  });

  describe('Query string validations', () => {
    const invalidQuery = 'CREATE TABLE users (LastName varchar(255))';
    const validQuery = 'SELECT * FROM users';

    it('calls onInvalidQuerySubmit when invalid', () => {
      createAceSpy();

      const invalidQuerySubmitSpy = createSpy();
      const component = mount(
        <NewQuery
          onInvalidQuerySubmit={invalidQuerySubmitSpy}
          onOsqueryTableSelect={noop}
          onTextEditorInputChange={noop}
          textEditorText={invalidQuery}
        />
      );

      component.find('.new-query__save-query-btn').simulate('click');

      const form = component.find('SaveQueryForm');
      const inputField = form.find('.save-query-form__input--name');

      fillInFormInput(inputField, 'my query');
      form.simulate('submit');

      expect(invalidQuerySubmitSpy).toHaveBeenCalledWith('Cannot INSERT or CREATE in osquery queries');
    });

    it('calls onNewQueryFormSubmit when valid', () => {
      createAceSpy();

      const onNewQueryFormSubmitSpy = createSpy();
      const component = mount(
        <NewQuery
          onNewQueryFormSubmit={onNewQueryFormSubmitSpy}
          onOsqueryTableSelect={noop}
          onTextEditorInputChange={noop}
          textEditorText={validQuery}
        />
      );

      component.find('.new-query__save-query-btn').simulate('click');

      const form = component.find('SaveQueryForm');
      const inputField = form.find('.save-query-form__input--name');

      fillInFormInput(inputField, 'my query');
      form.simulate('submit');

      expect(onNewQueryFormSubmitSpy).toHaveBeenCalled();
    });
  });
});

