import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import PlatformIcon from './PlatformIcon';

describe('PlatformIcon - component', () => {
  it('renders', () => {
    expect(mount(<PlatformIcon name="linux" />).length).toEqual(1);
  });

  it('renders text if no icon', () => {
    const component = mount(<PlatformIcon name="blah" />);

    expect(component.find('span').length).toEqual(1);
    expect(component.text()).toInclude('blah');
    expect(component.find('Icon').length).toEqual(0);
  });
});
