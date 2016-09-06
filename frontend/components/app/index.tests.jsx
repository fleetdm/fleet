import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import App from './index';

describe('App', () => {
  const component = mount(<App />);

  it('renders', () => {
    expect(component).toExist();
  });

  it('renders the appropriate text', () => {
    expect(component.text()).toInclude('If you can read this, React is rendering correctly!');
  });
});
