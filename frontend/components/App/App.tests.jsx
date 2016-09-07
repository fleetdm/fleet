import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { App } from './App';

describe('App - component', () => {
  const component = mount(<App />);

  it('renders', () => {
    expect(component).toExist();
  });

  it('renders the Style component', () => {
    expect(component.find('Style').length).toEqual(1);
  });
});
