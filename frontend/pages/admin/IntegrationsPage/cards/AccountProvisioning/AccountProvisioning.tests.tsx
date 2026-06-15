import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import createMockConfig from "__mocks__/configMock";

import AccountProvisioning from "./AccountProvisioning";

const configUrl = baseUrl("/config");

const defaultConfigHandler = http.get(configUrl, () =>
  HttpResponse.json(createMockConfig())
);

describe("AccountProvisioning", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  beforeEach(() => {
    mockServer.use(defaultConfigHandler);
  });

  it("renders the section heading", async () => {
    render(<AccountProvisioning />);
    await waitFor(() => {
      expect(screen.getByText("Account provisioning")).toBeInTheDocument();
    });
  });

  it("renders all three fields and the save button", async () => {
    render(<AccountProvisioning />);
    // Wait for spinner to disappear
    await waitFor(() => {
      expect(screen.getByLabelText(/token url/i)).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/client id/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/client secret/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("populates fields from API response", async () => {
    mockServer.use(
      http.get(configUrl, () =>
        HttpResponse.json(
          createMockConfig({
            apple_account_provisioning: {
              idp_token_url: "https://example.okta.com/oauth2/v1/token",
              idp_client_id: "my-client-id",
              oauth_idp_client_secret: "********",
            },
          })
        )
      )
    );

    render(<AccountProvisioning />);

    await waitFor(() => {
      expect(screen.getByLabelText(/token url/i)).toHaveValue(
        "https://example.okta.com/oauth2/v1/token"
      );
    });
    expect(screen.getByLabelText(/client id/i)).toHaveValue("my-client-id");
    expect(screen.getByLabelText(/client secret/i)).toHaveValue("********");
  });

  describe("Token URL validation", () => {
    it("shows a required error on blur when empty", async () => {
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(screen.getByLabelText(/token url/i)).toBeInTheDocument();
      });
      await user.click(screen.getByLabelText(/token url/i));
      await user.tab();
      await waitFor(() => {
        expect(screen.getByText(/token url is required/i)).toBeInTheDocument();
      });
    });

    it("shows an invalid URL error on blur when value is not a valid URL", async () => {
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(screen.getByLabelText(/token url/i)).toBeInTheDocument();
      });
      await user.type(screen.getByLabelText(/token url/i), "not-a-url");
      await user.tab();
      await waitFor(() => {
        expect(screen.getByText(/must be a valid url/i)).toBeInTheDocument();
      });
    });

    it("clears the error when a valid URL is entered", async () => {
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(screen.getByLabelText(/token url/i)).toBeInTheDocument();
      });
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
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(screen.getByLabelText(/client id/i)).toBeInTheDocument();
      });
      await user.click(screen.getByLabelText(/client id/i));
      await user.tab();
      await waitFor(() => {
        expect(screen.getByText(/client id is required/i)).toBeInTheDocument();
      });
    });
  });

  describe("Client secret validation", () => {
    it("shows a required error on blur when empty", async () => {
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(screen.getByLabelText(/client secret/i)).toBeInTheDocument();
      });
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
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(
          screen.getByRole("button", { name: /save/i })
        ).toBeInTheDocument();
      });
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
      const { user } = render(<AccountProvisioning />);
      await waitFor(() => {
        expect(screen.getByLabelText(/token url/i)).toBeInTheDocument();
      });
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
