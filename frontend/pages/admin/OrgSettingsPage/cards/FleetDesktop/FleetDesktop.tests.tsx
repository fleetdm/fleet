import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";

import FleetDesktop from "./FleetDesktop";

import { DEFAULT_TRANSPARENCY_URL } from "../constants";

const mdmWithIdPConfigured = createMockMdmConfig({
  end_user_authentication: {
    entity_id: "https://fleet.example.com",
    issuer_uri: "",
    metadata: "",
    metadata_url: "https://idp.example.com/metadata",
    idp_name: "Okta",
  },
});

describe("FleetDesktop", () => {
  const mockHandleSubmit = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("Rendering", () => {
    it("renders nothing when not premium tier", () => {
      const mockConfig = createMockConfig();

      const { container } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier={false}
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(container.firstChild).toBeNull();
    });

    it("renders the form when premium tier", () => {
      const mockConfig = createMockConfig();

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(screen.getByText("Fleet Desktop")).toBeInTheDocument();
      expect(
        screen.getByLabelText(/custom transparency url/i)
      ).toBeInTheDocument();
    });

    it("displays configured values from appConfig", () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "https://custom.example.com/transparency",
          alternative_browser_host: "browser.example.com",
          sso_enabled: false,
        },
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByDisplayValue("https://custom.example.com/transparency")
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("browser.example.com")
      ).toBeInTheDocument();
    });

    it("displays default transparency URL when none is configured", () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "",
          alternative_browser_host: "",
          sso_enabled: false,
        },
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByDisplayValue(DEFAULT_TRANSPARENCY_URL)
      ).toBeInTheDocument();
    });

    it("selects hourly token rotation by default", () => {
      const mockConfig = createMockConfig();

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByRole("group", { name: "End user authentication" })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).toBeChecked();
      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).not.toBeChecked();
    });

    it("selects single sign-on when sso_enabled is true", () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "",
          alternative_browser_host: "",
          sso_enabled: true,
        },
        mdm: mdmWithIdPConfigured,
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeChecked();
      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).not.toBeChecked();
    });

    it("keeps single sign-on selected when the IdP was cleared after saving", () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "",
          alternative_browser_host: "",
          sso_enabled: true,
        },
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const ssoRadio = screen.getByRole("radio", { name: /single sign-on/i });
      expect(ssoRadio).toBeChecked();
      expect(ssoRadio).toBeDisabled();
    });
  });

  describe("End user authentication", () => {
    it("disables single sign-on and explains why when no IdP is configured", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).toBeEnabled();

      await user.hover(screen.getByText("Single sign-on (SSO)"));

      await waitFor(() => {
        expect(
          screen.getByText(
            "This setting requires an IdP configured in Settings > Integrations > Authentication (SSO) > End users."
          )
        ).toBeInTheDocument();
      });
    });

    it("enables single sign-on when an IdP is configured", () => {
      const mockConfig = createMockConfig({ mdm: mdmWithIdPConfigured });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeEnabled();
    });

    it("disables single sign-on when the IdP is missing its metadata", () => {
      const mockConfig = createMockConfig({
        mdm: createMockMdmConfig({
          end_user_authentication: {
            entity_id: "https://fleet.example.com",
            issuer_uri: "",
            metadata: "",
            metadata_url: "",
            idp_name: "Okta",
          },
        }),
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeDisabled();
    });
  });

  describe("GitOps Mode", () => {
    it("disables inputs when gitops mode is enabled", () => {
      const mockConfig = createMockConfig({
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(screen.getByLabelText(/custom transparency url/i)).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeDisabled();
    });

    it("disables the single sign-on radio in gitops mode even when an IdP is configured", () => {
      const mockConfig = createMockConfig({
        mdm: mdmWithIdPConfigured,
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "",
          exceptions: { labels: false, software: false, secrets: true },
        },
      });

      renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeDisabled();
    });
  });

  describe("Form Validation", () => {
    it("shows error for invalid transparency URL without protocol", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/custom transparency url/i);
      await user.clear(input);
      await user.type(input, "invalid-url.com");
      await user.tab();

      await waitFor(() => {
        expect(
          screen.getByText(/custom transparency url must include protocol/i)
        ).toBeInTheDocument();
      });
    });

    it("accepts valid transparency URL with https protocol", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/custom transparency url/i);
      await user.clear(input);
      await user.type(input, "https://valid-url.com/transparency");
      await user.tab();

      await waitFor(() => {
        expect(
          screen.queryByText(/custom transparency url must include protocol/i)
        ).not.toBeInTheDocument();
      });
    });

    it("shows error for invalid browser host", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/browser host/i);
      await user.type(input, "not a valid hostname!");
      await user.tab();

      await waitFor(() => {
        expect(
          screen.getByText(/browser host must be a valid hostname/i)
        ).toBeInTheDocument();
      });
    });

    it("accepts valid browser hostname", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/browser host/i);
      await user.type(input, "fleet.example.com");
      await user.tab();

      await waitFor(() => {
        expect(
          screen.queryByText(/browser host must be a valid hostname/i)
        ).not.toBeInTheDocument();
      });
    });

    it("accepts hostnames with a port on alternative browser field", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/browser host/i);
      await user.type(input, "fleet.example.com:9809");
      await user.tab();

      await waitFor(() => {
        expect(
          screen.queryByText(/browser host must be a valid hostname/i)
        ).not.toBeInTheDocument();
      });
    });

    it("accepts IP addresses on browser alternative field", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/browser host/i);
      await user.type(input, "182.190.1.1:9809");
      await user.tab();

      await waitFor(() => {
        expect(
          screen.queryByText(/browser host must be a valid hostname/i)
        ).not.toBeInTheDocument();
      });
    });
  });

  describe("Form Submission", () => {
    it("calls handleSubmit with correct data structure", async () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "",
          alternative_browser_host: "",
          sso_enabled: false,
        },
      });

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const transparencyInput = screen.getByLabelText(
        /custom transparency url/i
      );
      const browserHostInput = screen.getByLabelText(/browser host/i);

      await user.clear(transparencyInput);
      await user.type(transparencyInput, "https://custom.example.com");
      await user.type(browserHostInput, "browser.example.com");

      const submitButton = screen.getByRole("button", { name: /save/i });
      await user.click(submitButton);

      expect(mockHandleSubmit).toHaveBeenCalledWith({
        fleet_desktop: {
          transparency_url: "https://custom.example.com",
          alternative_browser_host: "browser.example.com",
          sso_enabled: false,
        },
      });
    });

    it("submits sso_enabled when single sign-on is selected", async () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "https://custom.example.com",
          alternative_browser_host: "browser.example.com",
          sso_enabled: false,
        },
        mdm: mdmWithIdPConfigured,
      });

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      await user.click(screen.getByRole("radio", { name: /single sign-on/i }));

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeChecked();

      await user.click(screen.getByRole("button", { name: /save/i }));

      expect(mockHandleSubmit).toHaveBeenCalledWith({
        fleet_desktop: {
          transparency_url: "https://custom.example.com",
          alternative_browser_host: "browser.example.com",
          sso_enabled: true,
        },
      });
    });

    it("submits sso_enabled as false when switching back to token rotation", async () => {
      const mockConfig = createMockConfig({
        fleet_desktop: {
          transparency_url: "https://custom.example.com",
          alternative_browser_host: "browser.example.com",
          sso_enabled: true,
        },
        mdm: mdmWithIdPConfigured,
      });

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      await user.click(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      );

      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).toBeChecked();

      await user.click(screen.getByRole("button", { name: /save/i }));

      expect(mockHandleSubmit).toHaveBeenCalledWith({
        fleet_desktop: {
          transparency_url: "https://custom.example.com",
          alternative_browser_host: "browser.example.com",
          sso_enabled: false,
        },
      });
    });

    it("disables submit button when there are validation errors", async () => {
      const mockConfig = createMockConfig();

      const { user } = renderWithSetup(
        <FleetDesktop
          appConfig={mockConfig}
          handleSubmit={mockHandleSubmit}
          isPremiumTier
          isUpdatingSettings={false}
          router={createMockRouter()}
        />
      );

      const input = screen.getByLabelText(/custom transparency url/i);
      await user.clear(input);
      await user.type(input, "invalid-url");
      await user.tab();

      await waitFor(() => {
        const submitButton = screen.getByRole("button", { name: /save/i });
        expect(submitButton).toBeDisabled();
      });
    });
  });
});
