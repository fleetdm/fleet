import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { createAceSpy } from '../../../test/helpers';
import NewQuery from './index';

describe('NewQuery - component', () => {
  beforeEach(() => {
    createAceSpy();
  });
  afterEach(restoreSpies);

  it('renders the SaveQuerySection', () => {
    const component = mount(
      <NewQuery
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        textEditorText="Hello world"
      />
    );

    expect(component.find('SaveQuerySection').length).toEqual(1);
  });

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

  it('renders the SaveQueryForm', () => {
    const component = mount(
      <NewQuery
        onOsqueryTableSelect={noop}
        onTextEditorInputChange={noop}
        textEditorText="Hello world"
      />
    );

    component.find('Slider').simulate('click');

    expect(component.find('SaveQueryForm').length).toEqual(1);
  });

  describe('Query string validations', () => {
    const invalidQuery = 'CREATE TABLE users (LastName varchar(255))';
    const validQuery = 'SELECT * FROM users';

    it('calls onInvalidQuerySubmit when invalid', () => {
      const invalidQuerySubmitSpy = createSpy();
      const component = mount(
        <NewQuery
          onInvalidQuerySubmit={invalidQuerySubmitSpy}
          onOsqueryTableSelect={noop}
          onTextEditorInputChange={noop}
          textEditorText={invalidQuery}
        />
      );
      const form = component.find('SaveQueryForm');

      form.simulate('submit');

      expect(invalidQuerySubmitSpy).toHaveBeenCalledWith('Cannot INSERT or CREATE in osquery queries');
    });

    it('calls onNewQueryFormSubmit when valid', () => {
      const onNewQueryFormSubmitSpy = createSpy();
      const component = mount(
        <NewQuery
          onNewQueryFormSubmit={onNewQueryFormSubmitSpy}
          onOsqueryTableSelect={noop}
          onTextEditorInputChange={noop}
          textEditorText={validQuery}
        />
      );
      const form = component.find('SaveQueryForm');

      form.simulate('submit');

      expect(onNewQueryFormSubmitSpy).toHaveBeenCalled();
    });
  });
});

