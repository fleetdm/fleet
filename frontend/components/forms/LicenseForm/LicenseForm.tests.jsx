import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { fillInFormInput, itBehavesLikeAFormInputElement } from 'test/helpers';
import LicenseForm from 'components/forms/LicenseForm';

const defaultProps = {
  handleSubmit: noop,
};

describe('LicenseForm - component', () => {
  describe('rendering', () => {
    it('renders', () => {
      expect(mount(<LicenseForm {...defaultProps} />).length).toEqual(1);
    });
  });

  describe('license input', () => {
    const Form = mount(<LicenseForm {...defaultProps} />);

    it('renders an input field', () => {
      itBehavesLikeAFormInputElement(Form, 'license', 'textarea');
    });
  });

  describe('submitting the form', () => {
    afterEach(restoreSpies);

    it('calls the handleSubmit prop when valid', () => {
      const spy = createSpy();
      const props = { handleSubmit: spy };
      const Form = mount(<LicenseForm {...props} />);
      const LicenseField = Form.find({ name: 'license' }).find('textarea');
      const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';

      fillInFormInput(LicenseField, jwtToken);

      Form.simulate('submit');

      expect(spy).toHaveBeenCalledWith({ license: jwtToken });
    });

    it('does not submit when invalid', () => {
      const spy = createSpy();
      const props = { handleSubmit: spy };
      const Form = mount(<LicenseForm {...props} />);
      const LicenseField = Form.find({ name: 'license' }).find('textarea');
      const jwtToken = 'invalid.token';

      fillInFormInput(LicenseField, jwtToken);

      Form.simulate('submit');

      expect(spy).toNotHaveBeenCalled();
    });
  });
});
