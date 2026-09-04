import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import { IConfig, IEndUserAuthentication } from "interfaces/config";

import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";

import FleetDesktop from "./FleetDesktop";

import { DEFAULT_TRANSPARENCY_URL } from "../constants";

const mdmWithIdP = (idp?: Partial<IEndUserAuthentication>) =>
  createMockMdmConfig({
    end_user_authentication: {
      entity_id: "https://fleet.example.com",
      idp_name: "Okta",
      metadata: "",
      metadata_url: "https://idp.example.com/metadata",
      issuer_uri: "",
      ...idp,
    },
  });

describe("FleetDesktop", () => {
  const mockHandleSubmit = jest.fn();

  const renderCard = ({
    config,
    isPremiumTier = true,
  }: { config?: Partial<IConfig>; isPremiumTier?: boolean } = {}) =>
    renderWithSetup(
      <FleetDesktop
        appConfig={createMockConfig(config)}
        handleSubmit={mockHandleSubmit}
        isPremiumTier={isPremiumTier}
        isUpdatingSettings={false}
        router={createMockRouter()}
      />
    );

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("Rendering", () => {
    it("renders nothing when not premium tier", () => {
      const { container } = renderCard({ isPremiumTier: false });

      expect(container.firstChild).toBeNull();
    });

    it("renders the form when premium tier", () => {
      renderCard();

      expect(screen.getByText("Fleet Desktop")).toBeInTheDocument();
      expect(
        screen.getByLabelText(/custom transparency url/i)
      ).toBeInTheDocument();
    });

    it("displays configured values from appConfig", () => {
      renderCard({
        config: {
          fleet_desktop: {
            transparency_url: "https://custom.example.com/transparency",
            alternative_browser_host: "browser.example.com",
            sso_enabled: false,
          },
        },
      });

      expect(
        screen.getByDisplayValue("https://custom.example.com/transparency")
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("browser.example.com")
      ).toBeInTheDocument();
    });

    it("displays default transparency URL when none is configured", () => {
      renderCard({
        config: {
          fleet_desktop: {
            transparency_url: "",
            alternative_browser_host: "",
            sso_enabled: false,
          },
        },
      });

      expect(
        screen.getByDisplayValue(DEFAULT_TRANSPARENCY_URL)
      ).toBeInTheDocument();
    });

    it("selects hourly token rotation by default", () => {
      renderCard();

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
      renderCard({
        config: {
          fleet_desktop: {
            transparency_url: "",
            alternative_browser_host: "",
            sso_enabled: true,
          },
          mdm: mdmWithIdP(),
        },
      });

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeChecked();
      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).not.toBeChecked();
    });
  });

  describe("End user authentication", () => {
    // The SSO option needs an entity ID, an IdP name, and either inline
    // metadata or a metadata URL — dropping any one of them closes the gate.
    it.each([
      { idp: {}, expectEnabled: true, when: "the IdP is fully configured" },
      {
        idp: { metadata: "<EntityDescriptor />", metadata_url: "" },
        expectEnabled: true,
        when: "the IdP carries inline metadata instead of a URL",
      },
      {
        idp: { metadata: "", metadata_url: "" },
        expectEnabled: false,
        when: "the IdP has neither metadata nor a metadata URL",
      },
      {
        idp: { entity_id: "" },
        expectEnabled: false,
        when: "the IdP is missing its entity ID",
      },
      {
        idp: { idp_name: "" },
        expectEnabled: false,
        when: "the IdP is missing its name",
      },
      {
        idp: { entity_id: "", idp_name: "", metadata: "", metadata_url: "" },
        expectEnabled: false,
        when: "no IdP is configured",
      },
    ])(
      "single sign-on is enabled=$expectEnabled when $when",
      ({ idp, expectEnabled }) => {
        renderCard({ config: { mdm: mdmWithIdP(idp) } });

        const ssoRadio = screen.getByRole("radio", {
          name: /single sign-on/i,
        });
        if (expectEnabled) {
          expect(ssoRadio).toBeEnabled();
        } else {
          expect(ssoRadio).toBeDisabled();
        }
      }
    );

    it("explains how to configure an IdP when none is configured", async () => {
      const { user } = renderCard();

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
  });

  describe("GitOps Mode", () => {
    const gitOpsEnabled = {
      gitops_mode_enabled: true,
      repository_url: "",
      exceptions: { labels: false, software: false, secrets: true },
    };

    it("disables inputs when gitops mode is enabled", () => {
      renderCard({ config: { gitops: gitOpsEnabled } });

      expect(screen.getByLabelText(/custom transparency url/i)).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: /hourly token rotation/i })
      ).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeDisabled();
    });

    it("disables the single sign-on radio in gitops mode even when an IdP is configured", () => {
      renderCard({ config: { mdm: mdmWithIdP(), gitops: gitOpsEnabled } });

      expect(
        screen.getByRole("radio", { name: /single sign-on/i })
      ).toBeDisabled();
    });
  });

  describe("Form Validation", () => {
    it("shows error for invalid transparency URL without protocol", async () => {
      const { user } = renderCard();

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
      const { user } = renderCard();

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
      const { user } = renderCard();

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
      const { user } = renderCard();

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
      const { user } = renderCard();

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
      const { user } = renderCard();

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
      const { user } = renderCard({
        config: {
          fleet_desktop: {
            transparency_url: "",
            alternative_browser_host: "",
            sso_enabled: false,
          },
        },
      });

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
      const { user } = renderCard({
        config: {
          fleet_desktop: {
            transparency_url: "https://custom.example.com",
            alternative_browser_host: "browser.example.com",
            sso_enabled: false,
          },
          mdm: mdmWithIdP(),
        },
      });

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
      const { user } = renderCard({
        config: {
          fleet_desktop: {
            transparency_url: "https://custom.example.com",
            alternative_browser_host: "browser.example.com",
            sso_enabled: true,
          },
          mdm: mdmWithIdP(),
        },
      });

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
      const { user } = renderCard();

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
