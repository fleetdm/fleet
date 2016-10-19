import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import Footer from './Footer';

describe('Footer - component', () => {
  it('renders', () => {
    expect(mount(<Footer />)).toExist();
  });
});
