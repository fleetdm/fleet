import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import feetLogo from '../../../../assets/images/fleet-logo.svg';
import OrgLogoIcon from './OrgLogoIcon';

describe('OrgLogoIcon - component', () => {
  it('renders the Kolide Logo by default', () => {
    const component = mount(<OrgLogoIcon />);

    expect(component.state('imageSrc')).toEqual(feetLogo);
  });

  it('renders the image source when it is valid', () => {
    const component = mount(<OrgLogoIcon src="/assets/images/avatar.svg" />);

    expect(component.state('imageSrc')).toEqual('/assets/images/avatar.svg');
  });
});
