import expect from 'expect';

import validate from 'components/forms/admin/AppConfigForm/validate';

describe('AppConfigForm - validations', () => {
  const validFormData = {
    org_name: 'The Gnar Co.',
    authentication_type: 'username_password',
    kolide_server_url: 'https://gnar.dog',
    license: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ',
    sender_address: 'hi@gnar.dog',
    server: '192.168.99.100',
    port: 1025,
    user_name: 'gnardog',
    password: 'p@ssw0rd',
  };

  it('returns a valid object when the form data is valid', () => {
    expect(validate(validFormData)).toEqual({ valid: true, errors: {} });
  });

  it('validates presense of the license field', () => {
    const invalidFormData = {
      ...validFormData,
      license: '',
    };

    expect(validate(invalidFormData)).toEqual({
      valid: false,
      errors: {
        license: 'License must be present',
      },
    });
  });

  it('validates the license is a JWT token', () => {
    const invalidFormData = {
      ...validFormData,
      license: 'KFBR392',
    };

    expect(validate(invalidFormData)).toEqual({
      valid: false,
      errors: {
        license: 'License is not a valid JWT token',
      },
    });
  });

  it('validates presence of the org_name field', () => {
    const invalidFormData = {
      ...validFormData,
      org_name: '',
    };

    expect(validate(invalidFormData)).toEqual({
      valid: false,
      errors: {
        org_name: 'Organization Name must be present',
      },
    });
  });

  it('validates presence of the kolide_server_url field', () => {
    const invalidFormData = {
      ...validFormData,
      kolide_server_url: '',
    };

    expect(validate(invalidFormData)).toEqual({
      valid: false,
      errors: {
        kolide_server_url: 'Kolide Server URL must be present',
      },
    });
  });

  describe('smtp configurations', () => {
    it('validates the sender address', () => {
      const invalidFormData = {
        ...validFormData,
        sender_address: '',
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          sender_address: 'SMTP Sender Address must be present',
        },
      });
    });

    it('validates the smtp server', () => {
      const invalidFormData = {
        ...validFormData,
        server: '',
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          server: 'SMTP Server must be present',
        },
      });
    });

    it('validates the smtp server', () => {
      const invalidFormData = {
        ...validFormData,
        port: '',
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          server: 'SMTP Server Port must be present',
        },
      });
    });

    it('validates the password if auth type is not "none"', () => {
      const invalidFormData = {
        ...validFormData,
        password: '',
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          password: 'SMTP Password must be present',
        },
      });
    });

    it('validates the user_name if auth type is not "none"', () => {
      const invalidFormData = {
        ...validFormData,
        user_name: '',
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          user_name: 'SMTP Username must be present',
        },
      });
    });

    it('does not validate smtp config if only password is present and it is the fake password', () => {
      const formData = {
        ...validFormData,
        user_name: '',
        server: '',
        sender_address: '',
        password: '********',
      };
      const invalidFormData = {
        ...validFormData,
        user_name: '',
        server: '',
        sender_address: '',
        password: 'newPassword',
      };

      expect(validate(formData)).toEqual({
        valid: true,
        errors: {},
      });

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          sender_address: 'SMTP Sender Address must be present',
          server: 'SMTP Server must be present',
          user_name: 'SMTP Username must be present',
        },
      });
    });

    it('does not validate the user_name and password if the auth type is "none"', () => {
      const formData = {
        ...validFormData,
        authentication_type: 'authtype_none',
        password: '',
        user_name: '',
      };

      expect(validate(formData)).toEqual({ valid: true, errors: {} });
    });
  });
});
