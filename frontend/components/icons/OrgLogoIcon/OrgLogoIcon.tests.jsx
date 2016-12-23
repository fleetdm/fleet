import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import kolideLogo from '../../../../assets/images/kolide-logo.svg';
import OrgLogoIcon from './OrgLogoIcon';

describe('OrgLogoIcon - component', () => {
  it('renders the Kolide Logo by default', () => {
    const component = mount(<OrgLogoIcon />);

    expect(component.state('imageSrc')).toEqual(kolideLogo);
  });

  it('renders the image source when it is valid', () => {
    const component = mount(<OrgLogoIcon src="/assets/images/avatar.svg" />);

    expect(component.state('imageSrc')).toEqual('/assets/images/avatar.svg');
  });
});
