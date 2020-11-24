import React from 'react';
import { mount } from 'enzyme';

import Icon from './Icon';

describe('Icon - component', () => {
  it('renders', () => {
    expect(mount(<Icon name="success-check" />)).toBeTruthy();
  });
});
