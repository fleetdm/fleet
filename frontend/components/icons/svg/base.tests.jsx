import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import base from './base';
import { KolideLoginBackground } from './KolideLoginBackground/KolideLoginBackground.svg';

describe('base - svg HOC', () => {
  const WrappedComponent = base(KolideLoginBackground);
  const mountedComponent = mount(
    <WrappedComponent alt="image alt" fakeProp="fake" name="component name" />
  );

  it('renders a wrapped component', () => {
    expect(mountedComponent).toExist();
  });

  it('filters out unwanted props', () => {
    expect(mountedComponent.find(KolideLoginBackground).props()).toEqual({
      alt: 'image alt',
      name: 'component name',
      onClick: noop,
      variant: 'default',
      className: '',
    });
  });

  it('allows overriding the default variant prop', () => {
    const Component = base(KolideLoginBackground);
    const mounted = mount(
      <Component variant="my variant" />
    );

    expect(mounted.find(KolideLoginBackground).props()).toContain({
      variant: 'my variant',
    });
  });
});
