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

const createMockPolicy = (overrides?: Partial<IPolicy>): IPolicy => ({
  id: 1,
  name: "Test policy",
  query: "SELECT 1;",
  description: "",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@example.com",
  resolution: "",
  platform: "darwin",
  team_id: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  ...overrides,
});

const defaultProps = {
  onAddAutomation: jest.fn(),
  currentAutomatedPolicies: [] as number[],
};

describe("PolicyAutomations", () => {
  describe("CTA card (patch policy with patch_software, no install_software)", () => {
    it("shows the CTA card and Add automation button when canEditPolicy is true", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy()}
          canEditPolicy
          {...defaultProps}
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
          canEditPolicy
          onAddAutomation={onAddAutomation}
        />
      );

      await user.click(screen.getByRole("button", { name: /Add automation/ }));
      expect(onAddAutomation).toHaveBeenCalledTimes(1);
    });

    it("does NOT show the CTA card when canEditPolicy is false", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy()}
          canEditPolicy={false}
          {...defaultProps}
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
          canEditPolicy
          {...defaultProps}
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
          canEditPolicy
          {...defaultProps}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });

    it("hides the CTA card when patch_software is not set", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy({ patch_software: undefined })}
          canEditPolicy
          {...defaultProps}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });

    it("shows the CTA card for a no-team policy (team_id === 0)", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPatchPolicy({ team_id: 0 })}
          canEditPolicy
          {...defaultProps}
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
          canEditPolicy
          {...defaultProps}
        />
      );

      expect(screen.queryByText(/Automatically patch/)).not.toBeInTheDocument();
    });
  });

  describe("automations list", () => {
    it("shows empty state when no automations are configured", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy()}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.getByText("No automations")).toBeInTheDocument();
    });

    it("shows software automation row", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({
            install_software: { name: "Zoom", software_title_id: 42 },
          })}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.getByText("Zoom")).toBeInTheDocument();
      expect(screen.queryByText("No automations")).not.toBeInTheDocument();
    });

    it("shows script automation row", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({
            run_script: { id: 1, name: "fix.sh" },
          })}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.getByText("fix.sh")).toBeInTheDocument();
    });

    it("shows calendar automation row", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({ calendar_events_enabled: true })}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.getByText("Maintenance window")).toBeInTheDocument();
    });

    it("shows conditional access automation row", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({
            conditional_access_enabled: true,
          })}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.getByText("Block single sign-on")).toBeInTheDocument();
    });

    it("shows 'Webhook' for other automation when otherAutomationType is webhook", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({ id: 1 })}
          currentAutomatedPolicies={[1]}
          canEditPolicy={false}
          onAddAutomation={jest.fn()}
          otherAutomationType="webhook"
        />
      );

      expect(screen.getByText("Webhook")).toBeInTheDocument();
      expect(screen.queryByText("Ticket")).not.toBeInTheDocument();
    });

    it("shows 'Ticket' for other automation when otherAutomationType is ticket", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({ id: 1 })}
          currentAutomatedPolicies={[1]}
          canEditPolicy={false}
          onAddAutomation={jest.fn()}
          otherAutomationType="ticket"
        />
      );

      expect(screen.getByText("Ticket")).toBeInTheDocument();
      expect(screen.queryByText("Webhook")).not.toBeInTheDocument();
    });

    it("shows 'Webhook or ticket' for other automation when otherAutomationType is not set", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({ id: 1 })}
          currentAutomatedPolicies={[1]}
          canEditPolicy={false}
          onAddAutomation={jest.fn()}
        />
      );

      expect(screen.getByText("Webhook or ticket")).toBeInTheDocument();
    });

    it("does not show row type labels", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({ calendar_events_enabled: true })}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.queryByText("Calendar")).not.toBeInTheDocument();
    });
  });

  describe("footer text", () => {
    it("shows default footer text when continuous_automations_enabled is not set", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy()}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(
        screen.getByText(
          "Automations run on a host's first failure, or when a host's response changes from pass to fail."
        )
      ).toBeInTheDocument();
    });

    it("shows continuous footer text when continuous_automations_enabled is true", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy({
            continuous_automations_enabled: true,
          })}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(
        screen.getByText(
          "Software and script automations run every time Fleet receives a failing response. All other automations run on a host's first failure, or when a host's response changes from pass to fail."
        )
      ).toBeInTheDocument();
    });

    it("shows footer text even in the empty state", () => {
      renderWithAppContext(
        <PolicyAutomations
          storedPolicy={createMockPolicy()}
          canEditPolicy={false}
          {...defaultProps}
        />
      );

      expect(screen.getByText("No automations")).toBeInTheDocument();
      expect(
        screen.getByText(
          "Automations run on a host's first failure, or when a host's response changes from pass to fail."
        )
      ).toBeInTheDocument();
    });
  });
});
