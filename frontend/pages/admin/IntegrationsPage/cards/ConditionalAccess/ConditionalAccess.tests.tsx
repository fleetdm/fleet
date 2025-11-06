import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import createMockConfig from "__mocks__/configMock";
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

const updateConfigHandler = http.patch(baseUrl("/config"), () => {
  return HttpResponse.json(
    createMockConfig({
      conditional_access: {
        microsoft_entra_tenant_id: "",
        microsoft_entra_connection_configured: false,
        okta_idp_id: "okta-idp-123",
        okta_assertion_consumer_service_url: "https://example.com/acs",
        okta_audience_uri: "https://example.com",
        okta_certificate: "cert-data",
      },
    })
  );
});

describe("Conditional access", () => {
  describe("Not configured", () => {
    it("Renders both integration cards when nothing is configured", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
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
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
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
      mockServer.use(triggerConditionalAccessHandler);
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      // Open modal
      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[1]);

      // Fill in tenant ID
      const input = screen.getByRole("textbox");
      await user.type(input, "abcdefg");

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
      expect(screen.getByText("System scope profile")).toBeInTheDocument();
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

    // TODO: Re-enable this test after implementing file upload functionality for certificate
    it.skip("Saves Okta configuration when submitting form", async () => {
      mockServer.use(updateConfigHandler);
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
          },
        },
      });

      const { user } = render(<ConditionalAccess />);

      // Open modal
      const connectButtons = screen.getAllByText("Connect");
      await user.click(connectButtons[0]);

      // Fill in all fields
      // Note: Modal now has read-only textareas at indices 0 and 1 (System/User scope profiles)
      // so the editable inputs start at index 2
      const inputs = screen.getAllByRole("textbox");
      await user.type(inputs[2], "okta-idp-123"); // IdP ID
      await user.type(inputs[3], "https://example.com/acs"); // ACS URL
      await user.type(inputs[4], "https://example.com"); // Audience URI
      // Certificate is now a file upload section, not a text input
      // TODO: Implement file upload functionality and test it

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

  describe("Confirming configured", () => {
    it("Renders a spinner when Entra tenant id is present but configuration not yet confirmed", () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "abcdefg",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
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

      expect(screen.getByTestId("spinner")).toBeVisible();
    });
  });

  describe("Configured", () => {
    it("Shows Entra as configured when connection is confirmed", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "abcdefg",
          microsoft_entra_connection_configured: true,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
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

      expect(
        screen.getByText("Microsoft Entra conditional access configured")
      ).toBeInTheDocument();
      // Should only have Delete button for Entra (no Edit button per Figma design)
      expect(screen.getByText("Delete")).toBeInTheDocument();
      expect(screen.queryByText("Edit")).not.toBeInTheDocument();
    });

    it("Shows Okta as configured when all Okta fields are present", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "okta-idp-123",
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
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

      expect(
        screen.getByText("Okta conditional access configured")
      ).toBeInTheDocument();
    });

    it("Shows both providers as configured when both are set up", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "abcdefg",
          microsoft_entra_connection_configured: true,
          okta_idp_id: "okta-idp-123",
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
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

      expect(
        screen.getByText("Okta conditional access configured")
      ).toBeInTheDocument();
      expect(
        screen.getByText("Microsoft Entra conditional access configured")
      ).toBeInTheDocument();
    });

    it("Shows delete confirmation modal when clicking Delete on Okta", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "okta-idp-123",
          okta_assertion_consumer_service_url: "https://example.com/acs",
          okta_audience_uri: "https://example.com",
          okta_certificate: "cert-data",
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

      // Should show configured state
      expect(
        screen.getByText("Okta conditional access configured")
      ).toBeInTheDocument();

      // Click Delete button (first one is for Okta)
      const deleteButton = screen.getAllByText("Delete")[0];
      await user.click(deleteButton);

      // Should show delete confirmation modal
      await waitFor(() => {
        expect(
          screen.getByText(/Fleet will be disconnected from Okta/)
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
  });
});
