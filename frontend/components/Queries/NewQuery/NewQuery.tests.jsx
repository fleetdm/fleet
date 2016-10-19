import React from 'react';
import expect, { restoreSpies } from 'expect';
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
});

