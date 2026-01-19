import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import FleetDesktop from "./FleetDesktop";

import { DEFAULT_TRANSPARENCY_URL } from "../constants";

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
  });

  describe("GitOps Mode", () => {
    it("disables inputs when gitops mode is enabled", () => {
      const mockConfig = createMockConfig({
        gitops: { gitops_mode_enabled: true, repository_url: "" },
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
