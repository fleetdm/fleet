import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import Checkbox from './Checkbox';

describe('Checkbox - component', () => {
  it('renders', () => {
    expect(mount(<Checkbox />)).toExist();
  });
});
