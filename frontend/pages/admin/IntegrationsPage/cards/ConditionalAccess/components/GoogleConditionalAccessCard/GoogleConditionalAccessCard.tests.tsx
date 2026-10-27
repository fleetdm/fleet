import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import React, { useState } from "react";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import { AppContext, initialState } from "context/app";
import { IConfig } from "interfaces/config";
import mockServer from "test/mock-server";
import { baseUrl } from "test/test-utils";

import GoogleConditionalAccessCard from "./GoogleConditionalAccessCard";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const CUSTOMER_ID = "C01234abc";
const ADMIN_EMAIL = "fleet-admin@example.com";
const SERVICE_ACCOUNT_KEY = {
  client_email: "sa@example.iam.gserviceaccount.com",
  private_key: "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
};

const GITOPS_ON = {
  gitops_mode_enabled: true,
  repository_url: "https://github.com/example/fleet-gitops",
  exceptions: { labels: false, software: false, secrets: true },
};

const notConnectedConfig = () => createMockConfig();

const CONNECTED_CONDITIONAL_ACCESS = {
  microsoft_entra_tenant_id: "",
  microsoft_entra_connection_configured: false,
  okta_idp_id: "",
  okta_assertion_consumer_service_url: "",
  okta_audience_uri: "",
  okta_certificate: "",
  google_customer_id: CUSTOMER_ID,
  google_connection_configured: true,
};

const connectedConfig = (overrides?: Partial<IConfig>) =>
  createMockConfig({
    conditional_access: CONNECTED_CONDITIONAL_ACCESS,
    integrations: {
      ...createMockConfig().integrations,
      google_conditional_access: [
        {
          customer_id: CUSTOMER_ID,
          admin_email: ADMIN_EMAIL,
          api_key_json: "********",
        },
      ],
    },
    ...overrides,
  });

// The card reads and writes the config through AppContext, so keep it in
// state to observe the switch to the connected view.
const renderCard = (initialConfig: IConfig, onDelete = jest.fn()) => {
  const Harness = () => {
    const [config, setConfig] = useState<IConfig | null>(initialConfig);
    return (
      <AppContext.Provider
        value={{ ...initialState, isPremiumTier: true, config, setConfig }}
      >
        <GoogleConditionalAccessCard onDelete={onDelete} />
      </AppContext.Provider>
    );
  };
  return { user: userEvent.setup(), ...render(<Harness />) };
};

const getCustomerIdField = () => screen.getByLabelText("Google customer ID");
const getAdminEmailField = () => screen.getByLabelText("Admin email");
const getKeyField = () => screen.getByLabelText("Service account key");

describe("GoogleConditionalAccessCard", () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  describe("Not connected", () => {
    it("renders the three fields and Save", () => {
      renderCard(notConnectedConfig());

      expect(screen.getByText("Google")).toBeInTheDocument();
      expect(getCustomerIdField()).toHaveAttribute("placeholder", CUSTOMER_ID);
      expect(getAdminEmailField()).toHaveAttribute("placeholder", ADMIN_EMAIL);
      expect(getKeyField()).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("shows the three inline errors on an empty submit and makes no request", async () => {
      let requested = false;
      mockServer.use(
        http.patch(baseUrl("/config"), () => {
          requested = true;
          return HttpResponse.json({});
        })
      );
      const { user } = renderCard(notConnectedConfig());

      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(
        await screen.findByText("Enter your Google customer ID")
      ).toBeInTheDocument();
      expect(screen.getByText("Enter an admin email")).toBeInTheDocument();
      expect(
        screen.getByText("Paste the service account key JSON")
      ).toBeInTheDocument();
      expect(requested).toBe(false);
    });

    it("shows an inline error, not a toast, for a key that isn't JSON", async () => {
      const { user } = renderCard(notConnectedConfig());

      await user.type(getCustomerIdField(), CUSTOMER_ID);
      await user.type(getAdminEmailField(), ADMIN_EMAIL);
      await user.type(getKeyField(), "not json");
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(
        await screen.findByText("Paste the service account key JSON")
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("sends the parsed key, toasts, and switches to the connected view", async () => {
      let patchBody: unknown;
      mockServer.use(
        http.patch(baseUrl("/config"), async ({ request }) => {
          patchBody = await request.json();
          return HttpResponse.json(connectedConfig());
        })
      );
      const { user } = renderCard(notConnectedConfig());

      await user.type(getCustomerIdField(), CUSTOMER_ID);
      await user.type(getAdminEmailField(), ADMIN_EMAIL);
      // Pasting keeps the newlines in the multiline JSON.
      await user.click(getKeyField());
      await user.paste(JSON.stringify(SERVICE_ACCOUNT_KEY, null, 2));
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(
        await screen.findByText(`Google customer ID: ${CUSTOMER_ID}`)
      ).toBeInTheDocument();
      expect(
        screen.getByText(`Admin email: ${ADMIN_EMAIL}`)
      ).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Delete" })).toBeEnabled();
      expect(patchBody).toEqual({
        integrations: {
          google_conditional_access: [
            {
              customer_id: CUSTOMER_ID,
              admin_email: ADMIN_EMAIL,
              api_key_json: SERVICE_ACCOUNT_KEY,
            },
          ],
        },
      });
      expect(notify.success).toHaveBeenCalledWith(
        "Successfully connected to Google."
      );
    });

    it("toasts the server reason on an integration-level 422 and keeps the form", async () => {
      const reason =
        "Couldn't connect to Google: googleapi: Error 403: Not Authorized to access this resource/api";
      mockServer.use(
        http.patch(baseUrl("/config"), () =>
          HttpResponse.json(
            {
              message: "Validation Failed",
              errors: [
                { name: "integrations.google_conditional_access", reason },
              ],
            },
            { status: 422 }
          )
        )
      );
      const { user } = renderCard(notConnectedConfig());

      await user.type(getCustomerIdField(), CUSTOMER_ID);
      await user.type(getAdminEmailField(), ADMIN_EMAIL);
      await user.click(getKeyField());
      await user.paste(JSON.stringify(SERVICE_ACCOUNT_KEY));
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(reason);
      });
      expect(getCustomerIdField()).toHaveValue(CUSTOMER_ID);
      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("renders field-level 422s inline", async () => {
      mockServer.use(
        http.patch(baseUrl("/config"), () =>
          HttpResponse.json(
            {
              message: "Validation Failed",
              errors: [
                {
                  name: "integrations.google_conditional_access.customer_id",
                  reason:
                    'customer_id must be a Google customer ID starting with "C"',
                },
                {
                  name:
                    "integrations.google_conditional_access.api_key_json.private_key",
                  reason: "private_key is required",
                },
              ],
            },
            { status: 422 }
          )
        )
      );
      const { user } = renderCard(notConnectedConfig());

      await user.type(getCustomerIdField(), "01234abc");
      await user.type(getAdminEmailField(), ADMIN_EMAIL);
      await user.click(getKeyField());
      await user.paste(JSON.stringify({ client_email: "sa@example.com" }));
      await user.click(screen.getByRole("button", { name: "Save" }));

      expect(
        await screen.findByText("Enter a valid Google customer ID")
      ).toBeInTheDocument();
      expect(
        screen.getByText("Paste a valid service account key JSON")
      ).toBeInTheDocument();
    });

    it("keeps the stored key when an unconnected entry is resaved without a new key", async () => {
      let patchBody: unknown;
      mockServer.use(
        http.patch(baseUrl("/config"), async ({ request }) => {
          patchBody = await request.json();
          return HttpResponse.json(connectedConfig());
        })
      );
      const { user } = renderCard(
        connectedConfig({
          conditional_access: {
            ...CONNECTED_CONDITIONAL_ACCESS,
            google_connection_configured: false,
          },
        })
      );

      expect(getCustomerIdField()).toHaveValue(CUSTOMER_ID);
      expect(getKeyField()).toHaveValue("********");
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => {
        expect(patchBody).toEqual({
          integrations: {
            google_conditional_access: [
              { customer_id: CUSTOMER_ID, admin_email: ADMIN_EMAIL },
            ],
          },
        });
      });
    });
  });

  describe("Connected", () => {
    it("shows the customer ID and admin email and calls onDelete", async () => {
      const onDelete = jest.fn();
      const { user } = renderCard(connectedConfig(), onDelete);

      expect(
        screen.getByText(`Google customer ID: ${CUSTOMER_ID}`)
      ).toBeInTheDocument();
      expect(
        screen.getByText(`Admin email: ${ADMIN_EMAIL}`)
      ).toBeInTheDocument();
      expect(screen.queryByLabelText("Google customer ID")).toBeNull();

      await user.click(screen.getByRole("button", { name: "Delete" }));
      expect(onDelete).toHaveBeenCalledTimes(1);
    });
  });

  describe("GitOps mode", () => {
    it("disables the fields and Save", () => {
      renderCard(createMockConfig({ gitops: GITOPS_ON }));

      expect(getCustomerIdField()).toBeDisabled();
      expect(getAdminEmailField()).toBeDisabled();
      expect(getKeyField()).toBeDisabled();
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("disables Delete", () => {
      renderCard(connectedConfig({ gitops: GITOPS_ON }));

      expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
    });
  });
});
