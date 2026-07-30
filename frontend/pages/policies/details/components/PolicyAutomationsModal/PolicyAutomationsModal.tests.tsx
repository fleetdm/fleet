import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { IPolicy } from "interfaces/policy";

import PolicyAutomationsModal from "./PolicyAutomationsModal";

// Stub SoftwareIcon to avoid asset resolution in tests.
jest.mock("pages/SoftwarePage/components/icons/SoftwareIcon", () => {
  return () => <span data-testid="software-icon" />;
});

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

describe("PolicyAutomationsModal", () => {
  it("renders the automations list content", () => {
    render(
      <PolicyAutomationsModal
        storedPolicy={createMockPolicy({ calendar_events_enabled: true })}
        currentAutomatedPolicies={[]}
        onClose={jest.fn()}
      />
    );

    expect(screen.getByText("Maintenance window")).toBeInTheDocument();
  });

  it("renders the continuous-automations footer when continuous_automations_enabled is true", () => {
    render(
      <PolicyAutomationsModal
        storedPolicy={createMockPolicy({
          continuous_automations_enabled: true,
        })}
        currentAutomatedPolicies={[]}
        onClose={jest.fn()}
      />
    );

    expect(
      screen.getByText(/Software and script automations run/)
    ).toBeInTheDocument();
    expect(screen.getByText("every time")).toBeInTheDocument();
    expect(
      screen.queryByText(/Automations run on a host's first failure/)
    ).not.toBeInTheDocument();
  });

  it("forwards otherAutomationType to the automations list", () => {
    render(
      <PolicyAutomationsModal
        storedPolicy={createMockPolicy({ id: 1 })}
        currentAutomatedPolicies={[1]}
        otherAutomationType="ticket"
        onClose={jest.fn()}
      />
    );

    expect(screen.getByText("Ticket")).toBeInTheDocument();
    expect(screen.queryByText("Webhook or ticket")).not.toBeInTheDocument();
  });

  it("falls back to the generic other-automation label when otherAutomationType is not set", () => {
    render(
      <PolicyAutomationsModal
        storedPolicy={createMockPolicy({ id: 1 })}
        currentAutomatedPolicies={[1]}
        onClose={jest.fn()}
      />
    );

    expect(screen.getByText("Webhook or ticket")).toBeInTheDocument();
  });

  it("calls onClose when the Done button is clicked", async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    render(
      <PolicyAutomationsModal
        storedPolicy={createMockPolicy()}
        currentAutomatedPolicies={[]}
        onClose={onClose}
      />
    );

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
