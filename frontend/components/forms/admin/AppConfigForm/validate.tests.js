import expect from 'expect';

import validate from 'components/forms/admin/AppConfigForm/validate';

describe('AppConfigForm - validations', () => {
  const validFormData = {
    org_name: 'The Gnar Co.',
    authentication_type: 'username_password',
    kolide_server_url: 'https://gnar.dog',
    sender_address: 'hi@gnar.dog',
    enable_smtp: true,
    server: '192.168.99.100',
    port: '1025',
    user_name: 'gnardog',
    password: 'p@ssw0rd',
  };

  it('returns a valid object when the form data is valid', () => {
    expect(validate(validFormData)).toEqual({ valid: true, errors: {} });
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
        kolide_server_url: 'Fleet Server URL must be present',
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

    it('does not validate smtp config if smtp not enabled', () => {
      const formData = {
        ...validFormData,
        enable_smtp: false,
        user_name: '',
        server: '',
        sender_address: '',
        password: '********',
        port: '587',
      };
      const invalidFormData = {
        ...validFormData,
        user_name: '',
        server: '',
        sender_address: '',
        password: 'newPassword',
        port: '587',
      };
      const missingPortFormData = {
        ...validFormData,
        port: '',
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

      expect(validate(missingPortFormData)).toEqual({
        valid: false,
        errors: {
          server: 'SMTP Server Port must be present',
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
