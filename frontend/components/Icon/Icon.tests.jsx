import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import Icon from './Icon';

describe('Icon - component', () => {
  it('renders', () => {
    expect(mount(<Icon name="success-check" />)).toExist();
  });
});
