import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import ShadowBoxInput from './ShadowBoxInput';

describe('ShadowBoxInput - component', () => {
  it('renders an input field', () => {
    const component = mount(
      <ShadowBoxInput name="my-input" />
    );

    expect(component.find('input').length).toEqual(1);
  });

  it('does not render an icon without an icon class', () => {
    const component = mount(
      <ShadowBoxInput name="my-input" />
    );

    expect(component.find('i').length).toEqual(0);
  });

  it('renders an icon with an icon class', () => {
    const component = mount(
      <ShadowBoxInput name="my-input" iconClass="kolidecon-label" />
    );

    expect(component.find('i').length).toEqual(1);
  });
});
