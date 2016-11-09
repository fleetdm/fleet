import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import Dropdown from './Dropdown';

describe('Dropdown - component', () => {
  const options = [
    { text: 'Users', value: 'users' },
    { text: 'Groups', value: 'groups' },
  ];

  const props = {
    options,
  };

  it('renders the dropdown', () => {
    const component = mount(<Dropdown {...props} />);
    const dropdownSelect = component.find('Select');

    expect(dropdownSelect).toExist();
  });
});
