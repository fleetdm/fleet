import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AppContext, initialState } from "context/app";
import { IPolicy } from "interfaces/policy";
import createMockConfig from "__mocks__/configMock";
import PatchAutomationCta from "./PatchAutomationCta";

const createMockPatchPolicy = (overrides?: Partial<IPolicy>): IPolicy => ({
  id: 10,
  name: "macOS - Zoom up to date",
  query: "SELECT 1;",
  description: "Checks Zoom is up to date",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@example.com",
  resolution: "Install the latest version from self-service.",
  platform: "darwin",
  team_id: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "patch",
  patch_software: {
    name: "Zoom",
    display_name: "Zoom",
    software_title_id: 42,
  },
  ...overrides,
});

// Wrap with AppContext so GitOpsModeTooltipWrapper's useGitOpsMode hook works
const renderWithAppContext = (ui: React.ReactElement) => {
  return render(
    <AppContext.Provider
      value={{ ...initialState, config: createMockConfig() }}
    >
      {ui}
    </AppContext.Provider>
  );
};

describe("PatchAutomationCta", () => {
  describe("renders when conditions are met (patch policy with patch_software, no install_software)", () => {
    it("shows the CTA card and Add automation button when canEditPolicy is true", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy()}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.getByText(/Automatically patch Zoom/)).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Add automation/ })
      ).toBeInTheDocument();
    });

    it("prefers patch_software.display_name over name in the label", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy({
            patch_software: {
              name: "Microsoft.CompanyPortal",
              display_name: "Company Portal (Corp)",
              software_title_id: 42,
            },
          })}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(
        screen.getByText(/Automatically patch Company Portal \(Corp\)/)
      ).toBeInTheDocument();
    });

    it("normalizes well-known patch_software names when display_name is absent", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy({
            patch_software: {
              name: "Microsoft.CompanyPortal",
              software_title_id: 42,
            },
          })}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(
        screen.getByText(/Automatically patch Company Portal/)
      ).toBeInTheDocument();
    });

    it("calls onAddAutomation when the button is clicked", async () => {
      const user = userEvent.setup();
      const onAddAutomation = jest.fn();
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy()}
          canEditPolicy
          onAddAutomation={onAddAutomation}
        />
      );

      await user.click(screen.getByRole("button", { name: /Add automation/ }));
      expect(onAddAutomation).toHaveBeenCalledTimes(1);
    });

    it("does NOT render when canEditPolicy is false", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy()}
          canEditPolicy={false}
          onAddAutomation={jest.fn()}
        />
      );

      expect(
        screen.queryByText(/Automatically patch Zoom/)
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: /Add automation/ })
      ).not.toBeInTheDocument();
    });

    it("shows 'Adding...' text when isAddingAutomation is true", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy()}
          canEditPolicy
          onAddAutomation={jest.fn()}
          isAddingAutomation
        />
      );

      expect(screen.getByText("Adding...")).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: /Add automation/ })
      ).not.toBeInTheDocument();
    });
  });

  describe("renders nothing when conditions are not met", () => {
    it("for a dynamic (non-patch) policy", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy({ type: "dynamic" })}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });

    it("when patch_software is not set", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy({ patch_software: undefined })}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });

    it("renders for a no-team policy (team_id === 0)", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy({ team_id: 0 })}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.getByText(/Automatically patch Zoom/)).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Add automation/ })
      ).toBeInTheDocument();
    });

    it("when install_software is already set", () => {
      renderWithAppContext(
        <PatchAutomationCta
          storedPolicy={createMockPatchPolicy({
            install_software: {
              name: "Zoom",
              software_title_id: 42,
            },
          })}
          canEditPolicy
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });
  });
});
