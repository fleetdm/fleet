import expect from 'expect';

import validate from 'components/forms/LicenseForm/validate';

describe('LicenseForm - validation', () => {
  it('is valid given a valid license', () => {
    const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';
    const { errors, valid } = validate({ license: jwtToken });

    expect(valid).toEqual(true);
    expect(errors).toEqual({});
  });

  it('is not valid if the license is blank', () => {
    const jwtToken = '';
    const { errors, valid } = validate({ license: jwtToken });

    expect(valid).toEqual(false);
    expect(errors).toEqual({ license: 'License must be present' });
  });

  it('is not valid if the license is invalid', () => {
    const jwtToken = 'KFBR392';
    const { errors, valid } = validate({ license: jwtToken });

    expect(valid).toEqual(false);
    expect(errors).toEqual({
      license: 'License syntax is not valid. Please ensure you have entered the entire license. Please contact support@kolide.co if you need assistance',
    });
  });
});
