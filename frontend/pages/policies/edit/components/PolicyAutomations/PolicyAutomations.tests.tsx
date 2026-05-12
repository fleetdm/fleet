import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AppContext, initialState } from "context/app";
import { IPolicy } from "interfaces/policy";
import createMockConfig from "__mocks__/configMock";
import PolicyAutomations from "./PolicyAutomations";

// Stub SoftwareIcon to avoid asset resolution in tests
jest.mock("pages/SoftwarePage/components/icons/SoftwareIcon", () => {
  return () => <span data-testid="software-icon" />;
});

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

describe("PolicyAutomations", () => {
  describe("CTA card (patch policy with patch_software, no install_software)", () => {
    it("shows the CTA card and Add automation button when onAddAutomation is provided", () => {
      const onAddAutomation = jest.fn();
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy()}
          currentAutomatedPolicies={[]}
          onAddAutomation={onAddAutomation}
        />
      );

      expect(screen.getByText(/Automatically patch Zoom/)).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Add automation/ })
      ).toBeInTheDocument();
    });

    it("calls onAddAutomation when the button is clicked", async () => {
      const user = userEvent.setup();
      const onAddAutomation = jest.fn();
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy()}
          currentAutomatedPolicies={[]}
          onAddAutomation={onAddAutomation}
        />
      );

      await user.click(screen.getByRole("button", { name: /Add automation/ }));
      expect(onAddAutomation).toHaveBeenCalledTimes(1);
    });

    it("does NOT show the CTA card when onAddAutomation is undefined (no edit access)", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy()}
          currentAutomatedPolicies={[]}
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
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy()}
          currentAutomatedPolicies={[]}
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

  describe("CTA card is hidden when conditions are not met", () => {
    it("hides the CTA card for a dynamic (non-patch) policy", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy({ type: "dynamic" })}
          currentAutomatedPolicies={[]}
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });

    it("hides the CTA card when patch_software is not set", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy({ patch_software: undefined })}
          currentAutomatedPolicies={[]}
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });

    it("shows the CTA card for a no-team policy (team_id === 0)", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy({ team_id: 0 })}
          currentAutomatedPolicies={[]}
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.getByText(/Automatically patch Zoom/)).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /Add automation/ })
      ).toBeInTheDocument();
    });

    it("hides the CTA card when install_software is already set", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy({
            install_software: {
              name: "Zoom",
              software_title_id: 42,
            },
          })}
          currentAutomatedPolicies={[]}
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });
  });
});
