import validate from "components/forms/admin/AppConfigForm/validate";

describe("AppConfigForm - validations", () => {
  const validFormData = {
    org_name: "The Gnar Co.",
    authentication_type: "username_password",
    server_url: "https://gnar.dog",
    sender_address: "hi@gnar.dog",
    enable_smtp: true,
    server: "192.168.99.100",
    port: "1025",
    user_name: "gnardog",
    password: "p@ssw0rd",
    host_expiry_enabled: true,
    host_expiry_window: "42",
  };

  it("returns a valid object when the form data is valid", () => {
    expect(validate(validFormData)).toEqual({ valid: true, errors: {} });
  });

  it("validates presence of the org_name field", () => {
    const invalidFormData = {
      ...validFormData,
      org_name: "",
    };

    expect(validate(invalidFormData)).toEqual({
      valid: false,
      errors: {
        org_name: "Organization Name must be present",
      },
    });
  });

  it("validates presence of the server_url field", () => {
    const invalidFormData = {
      ...validFormData,
      server_url: "",
    };

    expect(validate(invalidFormData)).toEqual({
      valid: false,
      errors: {
        server_url: "Fleet Server URL must be present",
      },
    });
  });

  describe("smtp configurations", () => {
    it("validates the sender address", () => {
      const invalidFormData = {
        ...validFormData,
        sender_address: "",
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          sender_address: "SMTP Sender Address must be present",
        },
      });
    });

    it("validates the smtp server", () => {
      const invalidFormData = {
        ...validFormData,
        server: "",
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          server: "SMTP Server must be present",
        },
      });
    });

    it("validates the smtp port", () => {
      const invalidFormData = {
        ...validFormData,
        port: "",
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          server: "SMTP Server Port must be present",
        },
      });
    });

    it('validates the password if auth type is not "none"', () => {
      const invalidFormData = {
        ...validFormData,
        password: "",
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          password: "SMTP Password must be present",
        },
      });
    });

    it('validates the user_name if auth type is not "none"', () => {
      const invalidFormData = {
        ...validFormData,
        user_name: "",
      };

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          user_name: "SMTP Username must be present",
        },
      });
    });

    it("does not validate smtp config if smtp not enabled", () => {
      const formData = {
        ...validFormData,
        enable_smtp: false,
        user_name: "",
        server: "",
        sender_address: "",
        password: "********",
        port: "587",
      };
      const invalidFormData = {
        ...validFormData,
        user_name: "",
        server: "",
        sender_address: "",
        password: "newPassword",
        port: "587",
      };
      const missingPortFormData = {
        ...validFormData,
        port: "",
      };

      expect(validate(formData)).toEqual({
        valid: true,
        errors: {},
      });

      expect(validate(invalidFormData)).toEqual({
        valid: false,
        errors: {
          sender_address: "SMTP Sender Address must be present",
          server: "SMTP Server must be present",
          user_name: "SMTP Username must be present",
        },
      });

      expect(validate(missingPortFormData)).toEqual({
        valid: false,
        errors: {
          server: "SMTP Server Port must be present",
        },
      });
    });

    it('does not validate the user_name and password if the auth type is "none"', () => {
      const formData = {
        ...validFormData,
        authentication_type: "authtype_none",
        password: "",
        user_name: "",
      };

      expect(validate(formData)).toEqual({ valid: true, errors: {} });
    });
  });

  describe("host expiry settings", () => {
    it("does not validate missing expiry window", () => {
      const formData = {
        ...validFormData,
      };
      delete formData.host_expiry_window;
      expect(validate(formData)).toEqual({
        valid: false,
        errors: {
          host_expiry_window: "Host Expiry Window must be a positive number",
        },
      });
    });

    it("does not validate NaN expiry window", () => {
      const formData = {
        ...validFormData,
        host_expiry_window: "abcd",
      };
      expect(validate(formData)).toEqual({
        valid: false,
        errors: {
          host_expiry_window: "Host Expiry Window must be a positive number",
        },
      });
    });

    it("does not validate negative expiry window", () => {
      const formData = {
        ...validFormData,
        host_expiry_window: "-21",
      };
      expect(validate(formData)).toEqual({
        valid: false,
        errors: {
          host_expiry_window: "Host Expiry Window must be a positive number",
        },
      });
    });
  });
});
