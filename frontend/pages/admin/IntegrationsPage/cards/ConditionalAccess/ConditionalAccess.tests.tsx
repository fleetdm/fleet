import {
  render as rtlRender,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import React, { useState } from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import { AppContext, IAppContext, initialState } from "context/app";
import { IConfig } from "interfaces/config";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import ConditionalAccess from "./ConditionalAccess";

const triggerConditionalAccessHandler = http.post(
  baseUrl("/conditional-access/microsoft"),
  () => {
    return HttpResponse.json({
      microsoft_authentication_url: "https://example.com",
    });
  }
);

const TEST_TENANT_ID = "abcdefg";
const TEST_OKTA_IDP_ID = "okta-idp-123";

const updateConfigHandler = http.patch(baseUrl("/config"), () => {
  return HttpResponse.json(
    createMockConfig({
      conditional_access: {
        microsoft_entra_tenant_id: "",
        microsoft_entra_connection_configured: false,
        okta_idp_id: TEST_OKTA_IDP_ID,
        okta_assertion_consumer_service_url: "https://example.com/acs",
        okta_audience_uri: "https://example.com",
        okta_certificate: "cert-data",
        bypass_disabled: false,
      },
    })
  );
});

// The Google card renders its own fields and Save button inline, so scope
// queries to the open modal or to the bypass form.
const modal = () =>
  within(document.querySelector(".modal__modal_container") as HTMLElement);
const endUserExperience = () =>
  within(
    document.querySelector(
      ".conditional-access__end-user-experience"
    ) as HTMLElement
  );

// Helper to create a config with empty conditional access settings
const createEmptyConditionalAccessConfig = () =>
  createMockConfig({
    conditional_access: {
      microsoft_entra_tenant_id: "",
      microsoft_entra_connection_configured: false,
      okta_idp_id: "",
      okta_assertion_consumer_service_url: "",
      okta_audience_uri: "",
      okta_certificate: "",
      bypass_disabled: false,
    },
  });

describe("Conditional access", () => {
  describe("Not configured", () => {
    it("Renders both integration cards when nothing is configured", () => {
      const mockConfig = createEmptyConditionalAccessConfig();

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      render(<ConditionalAccess />);

      expect(screen.getByText("Okta")).toBeInTheDocument();
      expect(
        screen.getByText("Connect Okta to enable conditional access.")
      ).toBeInTheDocument();
      expect(screen.getByText("Microsoft Entra")).toBeInTheDocument();
      expect(
        screen.getByText("Connect Entra to enable conditional access.")
      ).toBeInTheDocument();
      // Should have two Connect buttons
      expect(screen.getAllByText("Connect")).toHaveLength(2);
    });

    it("Opens the Entra modal when clicking Connect on Entra card", async () => {
      const mockConfig = createEmptyConditionalAccessConfig();

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      // Click the second Connect button (Microsoft Entra)
      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[1]);

      // Modal should open
      expect(
        screen.getByText("Microsoft Entra conditional access")
      ).toBeInTheDocument();
      expect(screen.getByText("Microsoft Entra tenant ID")).toBeInTheDocument();
    });

    it("Triggers Microsoft auth flow when submitting Entra modal", async () => {
      const mockConfig = createEmptyConditionalAccessConfig();

      mockServer.use(triggerConditionalAccessHandler);
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      // Open modal
      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[1]);

      // Fill in tenant ID
      const input = modal().getByRole("textbox");
      await user.type(input, TEST_TENANT_ID);

      // Submit form
      const saveButton = modal().getByRole("button", { name: "Save" });
      await user.click(saveButton);

      // Should show the "continue in new tab" message
      await waitFor(() => {
        expect(
          screen.getByText(
            /To complete your integration, follow the instructions in the other tab/
          )
        ).toBeInTheDocument();
      });
    });

    it("Opens the Okta modal when clicking Connect on Okta card", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      // Click the first Connect button (Okta)
      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[0]);

      // Modal should open with new Figma structure
      expect(screen.getByText("Okta conditional access")).toBeInTheDocument();
      // Check for new sections
      expect(
        screen.getByText("Identity provider (IdP) signature certificate")
      ).toBeInTheDocument();
      expect(screen.getByText("User scope profile")).toBeInTheDocument();
      // Check for input fields
      expect(screen.getByText("IdP ID")).toBeInTheDocument();
      expect(
        screen.getByText("Assertion consumer service URL")
      ).toBeInTheDocument();
      expect(screen.getByText("Audience URI")).toBeInTheDocument();
      // Check for certificate upload section
      expect(screen.getByText("Okta certificate")).toBeInTheDocument();
    });

    it("Saves Okta configuration when submitting form", async () => {
      const mockConfig = createEmptyConditionalAccessConfig();

      mockServer.use(updateConfigHandler);
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      // Open modal
      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[0]);

      // Wait for modal to open
      await waitFor(() => {
        expect(screen.getByText("Okta conditional access")).toBeInTheDocument();
      });

      // Fill in text fields. textboxes[0] is the read-only User scope profile.
      const textboxes = modal().getAllByRole("textbox");
      await user.type(textboxes[1], TEST_OKTA_IDP_ID); // IdP ID
      await user.type(textboxes[2], "https://example.com/acs"); // ACS URL
      await user.type(textboxes[3], "https://example.com"); // Audience URI

      // Upload certificate file
      const certificateContent = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKm7MA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
-----END CERTIFICATE-----`;

      const file = new File([certificateContent], "certificate.pem", {
        type: "application/x-pem-file",
      });

      const fileInput = document.querySelector(
        'input[type="file"]'
      ) as HTMLInputElement;
      await user.upload(fileInput, file);

      // Wait for file to be processed
      await waitFor(() => {
        expect(screen.getByText("certificate.pem")).toBeInTheDocument();
      });

      // Submit form
      const saveButton = modal().getByRole("button", { name: "Save" });
      await user.click(saveButton);

      // Should show success message and close modal
      await waitFor(() => {
        expect(
          screen.queryByText("Okta conditional access")
        ).not.toBeInTheDocument();
      });
    });

    it("Preserves the certificate's trailing newline on submit (OKTA_CERTIFICATE skipTrim)", async () => {
      const mockConfig = createEmptyConditionalAccessConfig();

      let capturedBody: {
        conditional_access?: { okta_certificate?: string };
      } | null = null;
      mockServer.use(
        http.patch(baseUrl("/config"), async ({ request }) => {
          capturedBody = (await request.json()) as typeof capturedBody;
          return HttpResponse.json(mockConfig);
        })
      );

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true, config: mockConfig } },
      });

      const { user } = render(<ConditionalAccess />);
      await user.click(screen.getAllByText("Connect")[0]);
      await waitFor(() => {
        expect(screen.getByText("Okta conditional access")).toBeInTheDocument();
      });

      const textboxes = modal().getAllByRole("textbox");
      await user.type(textboxes[1], TEST_OKTA_IDP_ID);
      await user.type(textboxes[2], "https://example.com/acs");
      await user.type(textboxes[3], "https://example.com");

      // Trailing newline after -----END CERTIFICATE----- is standard PEM output
      // (e.g. from openssl). skipTrim keeps it intact; without the exemption,
      // useFormValidation trims it and the backend may reject the blob.
      const certificateContent = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKm7MA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
-----END CERTIFICATE-----
`;
      const file = new File([certificateContent], "certificate.pem", {
        type: "application/x-pem-file",
      });
      const fileInput = document.querySelector(
        'input[type="file"]'
      ) as HTMLInputElement;
      await user.upload(fileInput, file);
      await waitFor(() => {
        expect(screen.getByText("certificate.pem")).toBeInTheDocument();
      });

      await user.click(modal().getByRole("button", { name: "Save" }));
      await waitFor(() => {
        expect(capturedBody?.conditional_access?.okta_certificate).toBe(
          certificateContent
        );
      });
    });
  });

  describe("Confirming configured", () => {
    it("Renders a spinner when Entra tenant id is present but configuration not yet confirmed", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: TEST_TENANT_ID,
          microsoft_entra_connection_configured: false,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
          bypass_disabled: false,
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      render(<ConditionalAccess />);

      // Spinner has a built-in anti-flash delay, so wait for it to appear.
      expect(await screen.findByTestId("spinner")).toBeVisible();
    });
  });

  describe("Configured", () => {
    it("Shows Entra as configured when connection is confirmed", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: TEST_TENANT_ID,
          microsoft_entra_connection_configured: true,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
          bypass_disabled: false,
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const entraCopy = screen.getByText(/Microsoft Entra/);
      expect(entraCopy).toBeInTheDocument();
      expect(
        screen.getByText(/conditional access connected/)
      ).toBeInTheDocument();

      await user.hover(entraCopy);
      await waitFor(() => {
        expect(screen.getByText(TEST_TENANT_ID)).toBeInTheDocument();
      });

      // Should only have Delete button for Entra (no Edit button per Figma design)
      expect(screen.getByText("Delete")).toBeInTheDocument();
      expect(screen.queryByText("Edit")).not.toBeInTheDocument();
    });

    it("Shows Okta as configured when all Okta fields are present", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: TEST_OKTA_IDP_ID,
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
          bypass_disabled: false,
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const oktaCopy = screen.getByText(/Okta/);
      expect(oktaCopy).toBeInTheDocument();
      expect(
        screen.getByText(/conditional access connected/)
      ).toBeInTheDocument();

      await user.hover(oktaCopy);
      await waitFor(() => {
        expect(screen.getByText(TEST_OKTA_IDP_ID)).toBeInTheDocument();
      });
    });

    it("Shows both providers as configured when both are set up", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: TEST_TENANT_ID,
          microsoft_entra_connection_configured: true,
          okta_idp_id: TEST_OKTA_IDP_ID,
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
          bypass_disabled: false,
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const entraCopy = screen.getByText(/Microsoft Entra/);
      expect(entraCopy).toBeInTheDocument();

      await user.hover(entraCopy);
      await waitFor(() => {
        expect(screen.getByText(TEST_TENANT_ID)).toBeInTheDocument();
      });

      const oktaCopy = screen.getByText(/Okta/);
      expect(oktaCopy).toBeInTheDocument();
      await user.hover(oktaCopy);
      await waitFor(() => {
        expect(screen.getByText(TEST_OKTA_IDP_ID)).toBeInTheDocument();
      });

      expect(
        screen.queryAllByText(/conditional access connected/)
      ).toHaveLength(2);
    });

    it("Shows delete confirmation modal when clicking Delete on Okta", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: TEST_OKTA_IDP_ID,
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
          bypass_disabled: false,
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const oktaCopy = screen.getByText(/Okta/);
      expect(oktaCopy).toBeInTheDocument();
      expect(
        screen.getByText(/conditional access connected/)
      ).toBeInTheDocument();

      await user.hover(oktaCopy);
      await waitFor(() => {
        expect(screen.getByText(TEST_OKTA_IDP_ID)).toBeInTheDocument();
      });

      // Click Delete button (first one is for Okta)
      const deleteButton = screen.getAllByText("Delete")[0];
      await user.click(deleteButton);

      // Should show delete confirmation modal
      await waitFor(() => {
        expect(
          screen.getByText(/Before you delete, first unblock all end users/)
        ).toBeInTheDocument();
      });

      // Modal should have Delete and Cancel buttons
      expect(
        screen.getAllByRole("button", { name: "Delete" }).length
      ).toBeGreaterThan(0);
      expect(
        screen.getByRole("button", { name: "Cancel" })
      ).toBeInTheDocument();
    });
  });

  describe("GitOps mode", () => {
    it("Disables bypass toggle, Save, and Okta Delete under GitOps; leaves Entra Delete enabled (Entra setup is DB-backed, not YAML)", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: TEST_TENANT_ID,
          microsoft_entra_connection_configured: true,
          okta_idp_id: TEST_OKTA_IDP_ID,
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
          bypass_disabled: false,
        },
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      render(<ConditionalAccess />);

      // Checkbox exposes aria-label from `name`; disabled lives on aria-disabled.
      const bypassCheckbox = screen.getByRole("checkbox", {
        name: "bypassDisabled",
      });
      expect(bypassCheckbox).toHaveAttribute("aria-disabled", "true");

      expect(
        endUserExperience().getByRole("button", { name: "Save" })
      ).toBeDisabled();

      // Okta renders first, Entra second.
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      expect(deleteButtons).toHaveLength(2);
      expect(deleteButtons[0]).toBeDisabled();
      expect(deleteButtons[1]).toBeEnabled();
    });

    it("Keeps bypass Save disabled when GitOps flips on with a dirty form (isolates the GitOps gate from the unchanged-form check)", async () => {
      // The previous test's Save assertion is reinforced by the unchanged-form guard
      // (initial bypassDisabled matches config.bypass_disabled). This one dirties the
      // form with GitOps off, then rerenders with GitOps on so bypassDisabled survives
      // via useState — the only remaining reason Save can be disabled is the new gate.
      const baseConditionalAccess = {
        microsoft_entra_tenant_id: "",
        microsoft_entra_connection_configured: false,
        okta_idp_id: TEST_OKTA_IDP_ID,
        okta_assertion_consumer_service_url: "https://example.com/acs",
        okta_audience_uri: "https://example.com",
        okta_certificate: "cert-data",
        bypass_disabled: false,
      };
      const gitopsOffConfig = createMockConfig({
        conditional_access: baseConditionalAccess,
        gitops: {
          gitops_mode_enabled: false,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });
      const gitopsOnConfig = createMockConfig({
        conditional_access: baseConditionalAccess,
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      const client = new QueryClient({
        defaultOptions: { queries: { retry: false, cacheTime: 0 } },
      });
      const Harness = ({ app }: { app: Partial<IAppContext> }) => (
        <QueryClientProvider client={client}>
          <AppContext.Provider value={{ ...initialState, ...app }}>
            <ConditionalAccess />
          </AppContext.Provider>
        </QueryClientProvider>
      );

      const user = userEvent.setup();
      const { rerender } = rtlRender(
        <Harness app={{ isPremiumTier: true, config: gitopsOffConfig }} />
      );

      // Dirty the form: toggle bypassDisabled away from config.bypass_disabled.
      await user.click(
        screen.getByRole("checkbox", { name: "bypassDisabled" })
      );
      expect(
        endUserExperience().getByRole("button", { name: "Save" })
      ).toBeEnabled();

      // Flip GitOps on. bypassDisabled state persists via useState across the rerender,
      // so the unchanged-form guard still evaluates false. Save must disable solely
      // because of the GitOps gate.
      rerender(
        <Harness app={{ isPremiumTier: true, config: gitopsOnConfig }} />
      );
      expect(
        endUserExperience().getByRole("button", { name: "Save" })
      ).toBeDisabled();
    });

    it("Leaves Connect buttons enabled under GitOps so users can view the modal", () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
          bypass_disabled: false,
        },
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      render(<ConditionalAccess />);

      const connectButtons = screen.getAllByRole("button", { name: "Connect" });
      expect(connectButtons).toHaveLength(2);
      connectButtons.forEach((btn) => expect(btn).toBeEnabled());
    });

    it("Disables the Okta modal inputs and Save button under GitOps", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
          bypass_disabled: false,
        },
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const connectButtons = screen.getAllByRole("button", { name: "Connect" });
      await user.click(connectButtons[0]);

      await waitFor(() => {
        expect(screen.getByText("Okta conditional access")).toBeInTheDocument();
      });

      expect(modal().getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("Leaves the Entra modal fully functional under GitOps (OAuth is the only path, YAML can't set the tenant)", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
          bypass_disabled: false,
        },
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: mockConfig,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const connectButtons = screen.getAllByRole("button", { name: "Connect" });
      await user.click(connectButtons[1]);

      await waitFor(() => {
        expect(
          screen.getByText("Microsoft Entra conditional access")
        ).toBeInTheDocument();
      });

      expect(modal().getByRole("textbox")).toBeEnabled();
      // Validation also disables Save when the field is empty, so type first to
      // isolate the GitOps gating assertion.
      await user.type(modal().getByRole("textbox"), "test-tenant-id");
      expect(modal().getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("Disables Okta delete confirmation if GitOps turns on while it's open", async () => {
      const baseConditionalAccess = {
        microsoft_entra_tenant_id: "",
        microsoft_entra_connection_configured: false,
        okta_idp_id: TEST_OKTA_IDP_ID,
        okta_assertion_consumer_service_url: "https://example.com/acs",
        okta_audience_uri: "https://example.com",
        okta_certificate: "cert-data",
        bypass_disabled: false,
      };
      const gitopsOffConfig = createMockConfig({
        conditional_access: baseConditionalAccess,
        gitops: {
          gitops_mode_enabled: false,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });
      const gitopsOnConfig = createMockConfig({
        conditional_access: baseConditionalAccess,
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      const client = new QueryClient({
        defaultOptions: { queries: { retry: false, cacheTime: 0 } },
      });
      const Harness = ({ app }: { app: Partial<IAppContext> }) => (
        <QueryClientProvider client={client}>
          <AppContext.Provider value={{ ...initialState, ...app }}>
            <ConditionalAccess />
          </AppContext.Provider>
        </QueryClientProvider>
      );

      const user = userEvent.setup();
      const { rerender } = rtlRender(
        <Harness app={{ isPremiumTier: true, config: gitopsOffConfig }} />
      );

      // Open the Okta delete confirmation while GitOps is off.
      const deleteButtons = screen.getAllByRole("button", { name: "Delete" });
      await user.click(deleteButtons[0]);
      await waitFor(() => {
        expect(
          screen.getByText(/Before you delete, first unblock all end users/)
        ).toBeInTheDocument();
      });

      // Flip GitOps on; both the card trigger and the open confirmation's
      // Delete must disable so the write can't bypass the card-level gate.
      rerender(
        <Harness app={{ isPremiumTier: true, config: gitopsOnConfig }} />
      );

      const deleteButtonsAfter = screen.getAllByRole("button", {
        name: "Delete",
      });
      deleteButtonsAfter.forEach((btn) => expect(btn).toBeDisabled());
    });
  });

  describe("Google", () => {
    const GOOGLE_CUSTOMER_ID = "C01234abc";
    const GOOGLE_ADMIN_EMAIL = "fleet-admin@example.com";

    const createGoogleConfig = (
      connected: boolean,
      overrides?: Partial<IConfig>
    ) => {
      const config = createEmptyConditionalAccessConfig();
      return createMockConfig({
        conditional_access: {
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
          bypass_disabled: false,
          microsoft_entra_tenant_id: TEST_TENANT_ID,
          microsoft_entra_connection_configured: true,
          google_customer_id: connected ? GOOGLE_CUSTOMER_ID : "",
          google_connection_configured: connected,
        },
        integrations: {
          ...config.integrations,
          google_conditional_access: connected
            ? [
                {
                  customer_id: GOOGLE_CUSTOMER_ID,
                  admin_email: GOOGLE_ADMIN_EMAIL,
                  api_key_json: "********",
                },
              ]
            : undefined,
        },
        ...overrides,
      });
    };

    const renderStateful = (initialConfig: IConfig) => {
      const client = new QueryClient({
        defaultOptions: { queries: { retry: false, cacheTime: 0 } },
      });
      const Harness = () => {
        const [config, setConfig] = useState<IConfig | null>(initialConfig);
        return (
          <QueryClientProvider client={client}>
            <AppContext.Provider
              value={{
                ...initialState,
                isPremiumTier: true,
                config,
                setConfig,
              }}
            >
              <ConditionalAccess />
            </AppContext.Provider>
          </QueryClientProvider>
        );
      };
      return { user: userEvent.setup(), ...rtlRender(<Harness />) };
    };

    afterEach(() => {
      jest.restoreAllMocks();
    });

    it("Renders the Google card form beside Okta and Entra", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: { isPremiumTier: true, config: createGoogleConfig(false) },
        },
      });

      render(<ConditionalAccess />);

      expect(screen.getByText("Google")).toBeInTheDocument();
      expect(screen.getByLabelText("Google customer ID")).toBeInTheDocument();
      expect(screen.getByLabelText("Admin email")).toBeInTheDocument();
      expect(screen.getByLabelText("Service account key")).toBeInTheDocument();
    });

    it("Deletes the connection through the modal and returns to the form", async () => {
      const successSpy = jest.spyOn(notify, "success");
      let deleteCalled = false;
      mockServer.use(
        http.delete(baseUrl("/conditional-access/google"), () => {
          deleteCalled = true;
          return HttpResponse.json({});
        }),
        http.get(baseUrl("/config"), () =>
          HttpResponse.json(createGoogleConfig(false))
        )
      );
      const { user } = renderStateful(createGoogleConfig(true));

      expect(
        screen.getByText(`Google customer ID: ${GOOGLE_CUSTOMER_ID}`)
      ).toBeInTheDocument();
      // Entra renders first, Google second.
      await user.click(screen.getAllByRole("button", { name: "Delete" })[1]);
      expect(
        modal().getByText(
          "Fleet will stop marking devices as managed in Google. Devices already marked stay managed."
        )
      ).toBeInTheDocument();

      await user.click(modal().getByRole("button", { name: "Delete" }));

      expect(
        await screen.findByLabelText("Google customer ID")
      ).toBeInTheDocument();
      expect(deleteCalled).toBe(true);
      expect(successSpy).toHaveBeenCalledWith(
        "Successfully disconnected from Google."
      );
    });

    it("Disables Google Delete under GitOps and leaves Entra Delete enabled", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: createGoogleConfig(true, {
              gitops: {
                gitops_mode_enabled: true,
                repository_url: "https://github.com/example/fleet-gitops",
                exceptions: { labels: false, software: false, secrets: true },
              },
            }),
          },
        },
      });

      render(<ConditionalAccess />);

      const [entraDelete, googleDelete] = screen.getAllByRole("button", {
        name: "Delete",
      });
      expect(entraDelete).toBeEnabled();
      expect(googleDelete).toBeDisabled();
    });

    it("Doesn't render the Google card on Fleet Free", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: { isPremiumTier: false, config: createGoogleConfig(false) },
        },
      });

      render(<ConditionalAccess />);

      expect(
        screen.getByText(/This feature is included in Fleet Premium/i)
      ).toBeInTheDocument();
      expect(screen.queryByText("Google")).not.toBeInTheDocument();
    });
  });

  describe("Premium tier", () => {
    it("Shows premium feature message when not premium tier", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: false,
          },
        },
      });

      render(<ConditionalAccess />);

      expect(
        screen.getByText(/This feature is included in Fleet Premium/i)
      ).toBeInTheDocument();
    });

    it("Renders the 'Conditional access' section header on Fleet Free", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: false,
          },
        },
      });

      render(<ConditionalAccess />);

      expect(
        screen.getByRole("heading", {
          level: 2,
          name: /Conditional access/i,
        })
      ).toBeInTheDocument();
    });
  });

  describe("Okta modal validation", () => {
    const renderOktaModal = async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: createEmptyConditionalAccessConfig(),
          },
        },
      });
      const utils = render(<ConditionalAccess />);
      const connectButtons = screen.getAllByText("Connect");
      await utils.user.click(connectButtons[0]);
      await waitFor(() => {
        expect(screen.getByText("Okta conditional access")).toBeInTheDocument();
      });
      return utils;
    };

    it("Shows no field errors on open", async () => {
      await renderOktaModal();

      expect(screen.queryByText("Enter IdP ID")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Enter assertion consumer service URL")
      ).not.toBeInTheDocument();
      expect(screen.queryByText("Enter audience URI")).not.toBeInTheDocument();
      expect(screen.queryByText("Upload certificate")).not.toBeInTheDocument();
    });

    it("Does not show an error while the user types and clears, before blur", async () => {
      const { user } = await renderOktaModal();

      // textboxes[0] is the readonly Editor; [1] is IdP ID.
      const textboxes = modal().getAllByRole("textbox");
      await user.type(textboxes[1], "x");
      await user.clear(textboxes[1]);

      expect(screen.queryByText("Enter IdP ID")).not.toBeInTheDocument();
    });

    it("Shows the IdP ID error on blur of a dirty-then-empty field", async () => {
      const { user } = await renderOktaModal();

      const textboxes = modal().getAllByRole("textbox");
      await user.type(textboxes[1], "x");
      await user.clear(textboxes[1]);
      // Blur by tabbing to the next field.
      await user.tab();

      expect(await screen.findByText("Enter IdP ID")).toBeInTheDocument();
    });

    it("Clears the IdP ID error on focus after it was shown", async () => {
      const { user } = await renderOktaModal();

      const textboxes = modal().getAllByRole("textbox");
      await user.type(textboxes[1], "x");
      await user.clear(textboxes[1]);
      await user.tab();
      await screen.findByText("Enter IdP ID");

      await user.click(textboxes[1]);

      expect(screen.queryByText("Enter IdP ID")).not.toBeInTheDocument();
    });

    it("Reveals every field's error on submit of an empty form", async () => {
      const { user } = await renderOktaModal();

      const saveButton = modal().getByRole("button", { name: "Save" });
      expect(saveButton).toBeEnabled();
      await user.click(saveButton);

      expect(await screen.findByText("Enter IdP ID")).toBeInTheDocument();
      expect(
        screen.getByText("Enter assertion consumer service URL")
      ).toBeInTheDocument();
      expect(screen.getByText("Enter audience URI")).toBeInTheDocument();
      expect(screen.getByText("Upload certificate")).toBeInTheDocument();
    });
  });

  describe("Entra modal validation", () => {
    it("Shows 'Enter tenant ID' when submitting with an empty tenant", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: createEmptyConditionalAccessConfig(),
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[1]);

      await waitFor(() => {
        expect(
          screen.getByText("Microsoft Entra conditional access")
        ).toBeInTheDocument();
      });

      const saveButton = modal().getByRole("button", { name: "Save" });
      expect(saveButton).toBeEnabled();
      await user.click(saveButton);

      expect(await screen.findByText("Enter tenant ID")).toBeInTheDocument();
    });

    it("Inlines the backend's microsoft_tenant_id GUID error on the tenant field", async () => {
      mockServer.use(
        http.post(baseUrl("/conditional-access/microsoft"), () => {
          return HttpResponse.json(
            {
              message: "Validation Failed",
              errors: [
                {
                  name: "microsoft_tenant_id",
                  reason: "must be a valid Microsoft Entra tenant ID (GUID)",
                },
              ],
            },
            { status: 422 }
          );
        })
      );

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            config: createEmptyConditionalAccessConfig(),
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[1]);
      await waitFor(() => {
        expect(
          screen.getByText("Microsoft Entra conditional access")
        ).toBeInTheDocument();
      });

      await user.type(modal().getByRole("textbox"), "not-a-guid");
      await user.click(modal().getByRole("button", { name: "Save" }));

      expect(
        await screen.findByText("Enter a valid tenant ID (GUID format)")
      ).toBeInTheDocument();
    });
  });
});
