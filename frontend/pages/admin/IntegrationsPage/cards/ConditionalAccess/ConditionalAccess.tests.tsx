import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import ConditionalAccess from "./ConditionalAccess";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

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

const TEST_APPLE_PROFILE = "<plist>apple</plist>";
const TEST_WINDOWS_SCEP_PROFILE = "<Replace>windows scep</Replace>";
const TEST_WINDOWS_BROWSER_PROFILE = "<Replace>windows browser</Replace>";

const appleProfileHandler = http.get(
  baseUrl("/conditional_access/idp/apple/profile"),
  () => HttpResponse.text(TEST_APPLE_PROFILE)
);

const windowsProfilesHandler = http.get(
  baseUrl("/conditional_access/idp/windows/profiles"),
  () =>
    HttpResponse.json({
      scep_profile: TEST_WINDOWS_SCEP_PROFILE,
      browser_profile: TEST_WINDOWS_BROWSER_PROFILE,
    })
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
      const input = screen.getByRole("textbox");
      await user.type(input, TEST_TENANT_ID);

      // Submit form
      const saveButton = screen.getByRole("button", { name: "Save" });
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
      expect(screen.getByText("Windows profile")).toBeInTheDocument();
      expect(screen.getByText("Windows browser profile")).toBeInTheDocument();
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

      // Fill in text fields
      // Note: First three textareas are the read-only profiles
      const textboxes = screen.getAllByRole("textbox");
      await user.type(textboxes[3], TEST_OKTA_IDP_ID); // IdP ID
      await user.type(textboxes[4], "https://example.com/acs"); // ACS URL
      await user.type(textboxes[5], "https://example.com"); // Audience URI

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
      const saveButton = screen.getByRole("button", { name: "Save" });
      await user.click(saveButton);

      // Should show success message and close modal
      await waitFor(() => {
        expect(
          screen.queryByText("Okta conditional access")
        ).not.toBeInTheDocument();
      });
    });
  });

  describe("Okta modal profiles", () => {
    const openOktaModal = async () => {
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
      await user.click(screen.getAllByText("Connect")[0]);
      return user;
    };

    // The read-only profile textareas come first, in this order.
    const getProfileFields = () => {
      const [apple, windowsScep, windowsBrowser] = screen.getAllByRole(
        "textbox"
      );
      return { apple, windowsScep, windowsBrowser };
    };

    beforeEach(() => {
      jest.clearAllMocks();
    });

    it("shows the macOS and Windows profiles and copies each one", async () => {
      mockServer.use(appleProfileHandler, windowsProfilesHandler);

      const user = await openOktaModal();

      await waitFor(() => {
        const { apple, windowsScep, windowsBrowser } = getProfileFields();
        expect(apple).toHaveValue(TEST_APPLE_PROFILE);
        expect(windowsScep).toHaveValue(TEST_WINDOWS_SCEP_PROFILE);
        expect(windowsBrowser).toHaveValue(TEST_WINDOWS_BROWSER_PROFILE);
      });

      const copyButtons = screen.getAllByRole("button", {
        name: "Copy to clipboard",
      });
      // Spy after render: user-event's setup() replaces navigator.clipboard
      const writeTextSpy = jest.spyOn(navigator.clipboard, "writeText");
      await user.click(copyButtons[0]);
      await user.click(copyButtons[1]);
      await user.click(copyButtons[2]);
      expect(writeTextSpy.mock.calls).toEqual([
        [TEST_APPLE_PROFILE],
        [TEST_WINDOWS_SCEP_PROFILE],
        [TEST_WINDOWS_BROWSER_PROFILE],
      ]);
    });

    it("shows the Windows error reason and keeps the macOS profile when the Windows request fails", async () => {
      mockServer.use(
        appleProfileHandler,
        http.get(baseUrl("/conditional_access/idp/windows/profiles"), () =>
          HttpResponse.json(
            {
              message: "Bad request",
              errors: [
                {
                  name: "base",
                  reason: "global enroll secret is not configured",
                },
              ],
            },
            { status: 400 }
          )
        )
      );

      await openOktaModal();

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(
          "Failed to load Windows profiles: global enroll secret is not configured",
          expect.anything()
        );
      });
      await waitFor(() => {
        expect(getProfileFields().apple).toHaveValue(TEST_APPLE_PROFILE);
      });
      const { windowsScep, windowsBrowser } = getProfileFields();
      expect(windowsScep).toHaveValue("");
      expect(windowsBrowser).toHaveValue("");
    });

    it("keeps the macOS profile error toast unchanged", async () => {
      mockServer.use(
        windowsProfilesHandler,
        http.get(baseUrl("/conditional_access/idp/apple/profile"), () =>
          HttpResponse.json(
            {
              message: "Bad request",
              errors: [{ name: "base", reason: "missing private key" }],
            },
            { status: 400 }
          )
        )
      );

      await openOktaModal();

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(
          "Failed to load Apple profile: missing private key",
          expect.anything()
        );
      });
      await waitFor(() => {
        expect(getProfileFields().windowsScep).toHaveValue(
          TEST_WINDOWS_SCEP_PROFILE
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
});
