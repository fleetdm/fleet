import React from "react";

import { screen } from "@testing-library/react";
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
    it("Renders the empty form when no tenant id is saved", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
          },
        },
      });

      render(<ConditionalAccess />);

      expect(screen.getByText("Microsoft Entra tenant ID")).toBeInTheDocument();
      expect(screen.getByRole("textbox")).toHaveValue("");
    });
    it("Renders the 'continue in new tab' screen when the form is submitted", async () => {
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

      const input = screen.getByRole("textbox");
      await user.type(input, "abcdefg");
      await user.click(screen.getByRole("button"));

      expect(
        screen.getByText(
          "To complete your integration, follow the instructions in the other tab, then refresh this page to verify."
        )
      ).toBeInTheDocument();
    });
  });
  describe("Confirming configured", () => {
    it("Renders a spinner when tenant id is present but configuation not yet confirmed", () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "abcdefg",
          microsoft_entra_connection_configured: false,
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
    it("Renders the 'configured' screen when tenant id is present and configuration is confirmed", async () => {
      const mockConfig = createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "abcdefg",
          microsoft_entra_connection_configured: true,
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
        screen.getByText("Microsoft Entra tenant ID:")
      ).toBeInTheDocument();
      expect(screen.getByText("Delete")).toBeInTheDocument();
    });
  });
});
