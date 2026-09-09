import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";
import createMockLicense from "__mocks__/licenseMock";
import configAPI from "services/entities/config";
import { notify } from "components/ToastNotification";
import { IAppConfigFormProps } from "pages/admin/OrgSettingsPage/cards/constants";

import AccountProvisioning from "./AccountProvisioning";

jest.mock("services/entities/config");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const defaultProps: IAppConfigFormProps = {
  appConfig: createMockConfig({
    license: createMockLicense({ tier: "premium" }),
  }),
  handleSubmit: jest.fn() as IAppConfigFormProps["handleSubmit"],
  router: createMockRouter(),
};

const savedConfigProps: IAppConfigFormProps = {
  ...defaultProps,
  appConfig: createMockConfig({
    license: createMockLicense({ tier: "premium" }),
    mdm: {
      ...createMockConfig().mdm,
      apple_account_provisioning: {
        oauth_idp_token_url: "https://example.okta.com/oauth2/v1/token",
        oauth_idp_client_id: "my-client-id",
        oauth_idp_client_secret: "********",
      },
    },
  }),
};

describe("AccountProvisioning", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("renders the section heading", () => {
    render(<AccountProvisioning {...defaultProps} />);
    expect(screen.getByText("Account provisioning")).toBeInTheDocument();
  });

  it("renders premium message on free tier", () => {
    render(
      <AccountProvisioning
        {...defaultProps}
        appConfig={createMockConfig({
          license: createMockLicense({ tier: "free" }),
        })}
      />
    );
    expect(
      screen.getByText(/This feature is included in Fleet Premium/i)
    ).toBeInTheDocument();
  });

  it("renders all three fields and the save button", () => {
    render(<AccountProvisioning {...defaultProps} />);
    expect(screen.getByLabelText(/token url/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/client id/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/client secret/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("populates fields from appConfig prop", () => {
    render(
      <AccountProvisioning
        {...defaultProps}
        appConfig={createMockConfig({
          license: createMockLicense({ tier: "premium" }),
          mdm: {
            ...createMockConfig().mdm,
            apple_account_provisioning: {
              oauth_idp_token_url: "https://example.okta.com/oauth2/v1/token",
              oauth_idp_client_id: "my-client-id",
              oauth_idp_client_secret: "********",
            },
          },
        })}
      />
    );

    expect(screen.getByLabelText(/token url/i)).toHaveValue(
      "https://example.okta.com/oauth2/v1/token"
    );
    expect(screen.getByLabelText(/client id/i)).toHaveValue("my-client-id");
    expect(screen.getByLabelText(/client secret/i)).toHaveValue("********");
  });

  describe("Token URL validation", () => {
    it("shows a required error on blur when empty", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.click(screen.getByLabelText(/token url/i));
      await user.tab();
      await waitFor(() => {
        expect(screen.getByText(/token url is required/i)).toBeInTheDocument();
      });
    });

    it("shows an invalid URL error on blur when value is not a valid URL", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.type(screen.getByLabelText(/token url/i), "not-a-url");
      await user.tab();
      await waitFor(() => {
        expect(
          screen.getByText(/must be a valid https url/i)
        ).toBeInTheDocument();
      });
    });

    it("shows an invalid URL error on blur when the URL is not https", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.type(
        screen.getByLabelText(/token url/i),
        "http://example.okta.com/oauth2/v1/token"
      );
      await user.tab();
      await waitFor(() => {
        expect(
          screen.getByText(/must be a valid https url/i)
        ).toBeInTheDocument();
      });
    });

    it("clears the error when a valid URL is entered", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.type(screen.getByLabelText(/token url/i), "not-a-url");
      await user.tab();
      await waitFor(() => {
        expect(
          screen.getByText(/must be a valid https url/i)
        ).toBeInTheDocument();
      });
      // After the error shows, FormField replaces the label text with the error
      // message, so we locate the input by its placeholder instead.
      const tokenUrlInput = screen.getByPlaceholderText(
        /yourdomain\.okta\.com/i
      );
      await user.clear(tokenUrlInput);
      await user.type(
        tokenUrlInput,
        "https://yourdomain.okta.com/oauth2/v1/token"
      );
      await waitFor(() => {
        expect(
          screen.queryByText(/must be a valid https url/i)
        ).not.toBeInTheDocument();
      });
    });
  });

  describe("Client ID validation", () => {
    it("shows a required error on blur when empty", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.click(screen.getByLabelText(/client id/i));
      await user.tab();
      await waitFor(() => {
        expect(screen.getByText(/client id is required/i)).toBeInTheDocument();
      });
    });
  });

  describe("Client secret validation", () => {
    it("shows a required error on blur when empty", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.click(screen.getByLabelText(/client secret/i));
      await user.tab();
      await waitFor(() => {
        expect(
          screen.getByText(/client secret is required/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe("Form submission", () => {
    it("shows all errors on submit when all fields are empty", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.click(screen.getByRole("button", { name: /save/i }));
      await waitFor(() => {
        expect(screen.getByText(/token url is required/i)).toBeInTheDocument();
        expect(screen.getByText(/client id is required/i)).toBeInTheDocument();
        expect(
          screen.getByText(/client secret is required/i)
        ).toBeInTheDocument();
      });
    });

    it("does not submit when token URL is invalid", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.type(screen.getByLabelText(/token url/i), "not-a-url");
      await user.type(screen.getByLabelText(/client id/i), "my-client-id");
      await user.type(
        screen.getByLabelText(/client secret/i),
        "my-client-secret"
      );
      await user.click(screen.getByRole("button", { name: /save/i }));
      await waitFor(() => {
        expect(
          screen.getByText(/must be a valid https url/i)
        ).toBeInTheDocument();
      });
      expect(configAPI.update).not.toHaveBeenCalled();
    });
  });

  describe("Editing the token URL of a saved configuration", () => {
    it("clears the masked client secret and flags it for re-entry", async () => {
      const { user } = render(<AccountProvisioning {...savedConfigProps} />);
      await user.type(screen.getByLabelText(/token url/i), "x");

      // the error replaces the "Client secret" label text
      const secretInput = screen.getByLabelText(
        /client secret must be re-entered/i
      );
      expect(secretInput).toHaveValue("");
    });

    it("does not clear a client secret the user has already re-entered", async () => {
      const { user } = render(<AccountProvisioning {...savedConfigProps} />);
      const secretInput = screen.getByLabelText(/client secret/i);
      await user.clear(secretInput);
      await user.type(secretInput, "new-secret");

      await user.type(screen.getByLabelText(/token url/i), "x");

      expect(secretInput).toHaveValue("new-secret");
      expect(
        screen.queryByText(/client secret must be re-entered/i)
      ).not.toBeInTheDocument();
    });

    it("keeps the masked client secret when only the client ID is edited", async () => {
      const { user } = render(<AccountProvisioning {...savedConfigProps} />);
      await user.type(screen.getByLabelText(/client id/i), "x");
      expect(screen.getByLabelText(/client secret/i)).toHaveValue("********");
    });

    it("blocks submission until the client secret is re-entered", async () => {
      const { user } = render(<AccountProvisioning {...savedConfigProps} />);
      await user.type(screen.getByLabelText(/token url/i), "x");
      await user.click(screen.getByRole("button", { name: /save/i }));
      await waitFor(() => {
        expect(
          screen.getByText(/client secret is required/i)
        ).toBeInTheDocument();
      });
      expect(configAPI.update).not.toHaveBeenCalled();

      // the error replaces the "Client secret" label text
      const secretInput = screen.getByLabelText(/client secret is required/i);
      await user.type(secretInput, "new-secret");
      await user.click(screen.getByRole("button", { name: /save/i }));

      await waitFor(() => {
        expect(configAPI.update).toHaveBeenCalledWith({
          mdm: {
            apple_account_provisioning: {
              oauth_idp_token_url: "https://example.okta.com/oauth2/v1/tokenx",
              oauth_idp_client_id: "my-client-id",
              oauth_idp_client_secret: "new-secret",
            },
          },
        });
      });
    });
  });

  describe("Server errors", () => {
    const fillValidForm = async (user: ReturnType<typeof render>["user"]) => {
      await user.type(
        screen.getByLabelText(/token url/i),
        "https://example.okta.com/oauth2/v1/token"
      );
      await user.type(screen.getByLabelText(/client id/i), "my-client-id");
      await user.type(screen.getByLabelText(/client secret/i), "my-secret");
      await user.click(screen.getByRole("button", { name: /save/i }));
    };

    it("surfaces field-level server errors inline on the matching fields and in the toast", async () => {
      (configAPI.update as jest.Mock).mockRejectedValue({
        status: 422,
        data: {
          message: "Validation Failed",
          errors: [
            {
              name: "mdm.apple_account_provisioning.oauth_idp_client_secret",
              reason:
                "oauth_idp_client_secret must be provided when changing oauth_idp_token_url",
            },
            {
              name: "mdm.apple_account_provisioning.oauth_idp_token_url",
              reason: "must be a valid https URL",
            },
          ],
        },
      });

      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await fillValidForm(user);

      await waitFor(() => {
        expect(
          screen.getByText(/must be provided when changing/i)
        ).toBeInTheDocument();
      });
      expect(
        screen.getByText(/must be a valid https url/i)
      ).toBeInTheDocument();
      expect(notify.error).toHaveBeenCalledWith(
        expect.stringContaining("must be provided when changing"),
        expect.anything()
      );
    });

    it("includes the server reason in the error toast for non-field errors", async () => {
      (configAPI.update as jest.Mock).mockRejectedValue({
        status: 422,
        data: {
          message: "Validation Failed",
          errors: [
            {
              name: "mdm.apple_account_provisioning",
              reason: "Missing required private key",
            },
          ],
        },
      });

      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await fillValidForm(user);

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(
          expect.stringContaining("Missing required private key"),
          expect.anything()
        );
      });
    });
  });
});
