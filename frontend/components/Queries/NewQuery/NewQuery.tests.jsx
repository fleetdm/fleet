import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import NewQuery from './index';

const createAceSpy = () => {
  return spyOn(global.window.ace, 'edit').andReturn({
    $options: {},
    getValue: () => { return 'Hello world'; },
    getSession: () => {
      return {
        getMarkers: noop,
        setAnnotations: noop,
        setMode: noop,
        setUseWrapMode: noop,
      };
    },
    handleOptions: noop,
    handleMarkers: noop,
    on: noop,
    renderer: {
      setShowGutter: noop,
    },
    session: {
      on: noop,
    },
    setFontSize: noop,
    setMode: noop,
    setOption: noop,
    setOptions: noop,
    setShowPrintMargin: noop,
    setTheme: noop,
    setValue: noop,
  });
};

describe('NewQuery - component', () => {
  beforeEach(createAceSpy);
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

