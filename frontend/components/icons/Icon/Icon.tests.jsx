import React from 'react';
import { mount } from 'enzyme';

import Icon from './Icon';

describe('Icon - component', () => {
  it('renders', () => {
    expect(mount(<Icon name="main-hosts" size="24" />)).toBeTruthy();
  });
});
