import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
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

  // Regression: pre-migration the port error was set to the placeholder
  // string "Port" and shared its message key with the server error, so
  // fixing one cleared the other. Each field now carries its own copy.
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

    it("reports only the missing field when the other is filled", () => {
      const errors = validate({ ...baseFormData, smtpPort: undefined });

      expect(errors.smtpPort).toBe("Enter a server port");
      expect(errors.smtpServer).toBeUndefined();
    });
  });
});
