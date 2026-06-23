import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";
import { IAppConfigFormProps } from "pages/admin/OrgSettingsPage/cards/constants";

import AccountProvisioning from "./AccountProvisioning";

const defaultProps: IAppConfigFormProps = {
  appConfig: createMockConfig(),
  handleSubmit: jest.fn() as IAppConfigFormProps["handleSubmit"],
  router: createMockRouter(),
};

describe("AccountProvisioning", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("renders the section heading", () => {
    render(<AccountProvisioning {...defaultProps} />);
    expect(screen.getByText("Account provisioning")).toBeInTheDocument();
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
        expect(screen.getByText(/must be a valid url/i)).toBeInTheDocument();
      });
    });

    it("clears the error when a valid URL is entered", async () => {
      const { user } = render(<AccountProvisioning {...defaultProps} />);
      await user.type(screen.getByLabelText(/token url/i), "not-a-url");
      await user.tab();
      await waitFor(() => {
        expect(screen.getByText(/must be a valid url/i)).toBeInTheDocument();
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
          screen.queryByText(/must be a valid url/i)
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
        expect(screen.getByText(/must be a valid url/i)).toBeInTheDocument();
      });
    });
  });
});
