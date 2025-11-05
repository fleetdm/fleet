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
