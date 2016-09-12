import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import LoginPage from './LoginPage';
import * as bgImageUtility from '../../utilities/backgroundImage';

describe('LoginPage - component', () => {
  beforeEach(() => {
    spyOn(bgImageUtility, 'loadBackground').andReturn(noop);
    spyOn(bgImageUtility, 'resizeBackground').andReturn(noop);
  });

  afterEach(restoreSpies);

  it('renders the LoginForm', () => {
    const page = mount(<LoginPage />);

    expect(page.find('LoginForm').length).toEqual(1);
  });

  it('render the Kolide Text logo', () => {
    const page = mount(<LoginPage />);

    expect(page.find('Icon').first().props()).toInclude({
      name: 'kolideText',
    });
  });
});
