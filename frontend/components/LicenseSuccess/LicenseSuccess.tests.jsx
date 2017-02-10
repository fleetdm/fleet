import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import { licenseStub } from 'test/stubs';
import LicenseSuccess from 'components/LicenseSuccess';

const defaultProps = {
  license: licenseStub(),
};

describe('LicenseSuccess - component', () => {
  describe('rendering', () => {
    it('renders', () => {
      expect(mount(<LicenseSuccess {...defaultProps} />).length).toEqual(1, 'Expected LicenseSuccess component to render');
    });
  });
});
