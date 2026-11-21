import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { trimFormData } from "hooks/useFormValidation";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import Smtp, { validate } from "./Smtp";

describe("Smtp", () => {
  const mockHandleSubmit = jest.fn().mockResolvedValue(true);

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the STARTTLS guidance on hover of the SSL/TLS checkbox label", async () => {
    const mockConfig = createMockConfig();

    const { user } = renderWithSetup(
      <Smtp
        appConfig={mockConfig}
        handleSubmit={mockHandleSubmit}
        isUpdatingSettings={false}
        router={createMockRouter()}
      />
    );

    const label = screen.getByText("Use SSL/TLS to connect (recommended)");
    await user.hover(label);

    await waitFor(() => {
      expect(
        screen.getByText(/STARTTLS must first be disabled in/i)
      ).toBeInTheDocument();
    });
  });

  // Server and port must carry independent error copy, not share a key.
  describe("validate", () => {
    const baseFormData = {
      enableSMTP: true,
      smtpSenderAddress: "admin@example.com",
      smtpServer: "smtp.example.com",
      smtpPort: 587,
      smtpEnableSSLTLS: true,
      smtpAuthenticationType: "authtype_none",
      smtpUsername: "",
      smtpPassword: "",
      smtpAuthenticationMethod: "",
    };

    it("reports server and port errors independently", () => {
      const errors = validate({
        ...baseFormData,
        smtpServer: "",
        smtpPort: undefined,
      });

      expect(errors.smtpServer).toBe("Enter an SMTP server");
      expect(errors.smtpPort).toBe("Enter a server port");
    });

    it("accepts a valid sender address with surrounding whitespace", () => {
      const errors = validate({
        ...baseFormData,
        smtpSenderAddress: " admin@example.com ",
      });

      expect(errors.smtpSenderAddress).toBeUndefined();
    });

    it("still rejects an invalid sender address", () => {
      const errors = validate({
        ...baseFormData,
        smtpSenderAddress: " not-an-email ",
      });

      expect(errors.smtpSenderAddress).toBe("Enter a valid sender address");
    });

    it("reports only the missing field when the other is filled", () => {
      const errors = validate({ ...baseFormData, smtpPort: undefined });

      expect(errors.smtpPort).toBe("Enter a server port");
      expect(errors.smtpServer).toBeUndefined();
    });

    // Blur validates the raw value; Save validates the trimmed one. A
    // whitespace-only value counts as empty and must get the same result from
    // both.
    it("treats a whitespace-only sender address as empty when SMTP is enabled", () => {
      const data = { ...baseFormData, smtpSenderAddress: "   " };

      expect(validate(data).smtpSenderAddress).toBe("Enter a sender address");
      expect(validate(data)).toEqual(validate(trimFormData(data)));
    });

    it("accepts a whitespace-only sender address when SMTP is disabled", () => {
      const data = {
        ...baseFormData,
        enableSMTP: false,
        smtpSenderAddress: "   ",
      };

      expect(validate(data).smtpSenderAddress).toBeUndefined();
      expect(validate(data)).toEqual(validate(trimFormData(data)));
    });

    it("treats whitespace-only server and username as empty", () => {
      const data = {
        ...baseFormData,
        smtpServer: "   ",
        smtpAuthenticationType: "authtype_username_password",
        smtpUsername: "   ",
        smtpPassword: "secret",
      };

      expect(validate(data).smtpServer).toBe("Enter an SMTP server");
      expect(validate(data).smtpUsername).toBe("Enter an SMTP username");
      expect(validate(data)).toEqual(
        validate(trimFormData(data, ["smtpPassword"]))
      );
    });
  });
});
