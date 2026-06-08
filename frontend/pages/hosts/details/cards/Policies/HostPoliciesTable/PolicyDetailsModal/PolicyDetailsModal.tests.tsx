import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { renderWithSetup } from "test/test-utils";

import { IHostPolicy } from "interfaces/policy";

import PolicyDetailsModal from "./PolicyDetailsModal";

const createMockPolicy = (
  overrides: Partial<IHostPolicy> = {}
): IHostPolicy => ({
  id: 1,
  name: "Test policy",
  query: "SELECT 1;",
  description: "",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@fleet.co",
  resolution: "",
  platform: "",
  team_id: null,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "policy",
  response: "fail",
  ...overrides,
});

describe("PolicyDetailsModal", () => {
  it("renders policy name, description, and resolution", () => {
    renderWithSetup(
      <PolicyDetailsModal
        onCancel={noop}
        policy={createMockPolicy({
          name: "Disk encryption enabled",
          description: "Checks that FileVault is enabled.",
          resolution: "Enable FileVault in System Settings.",
        })}
      />
    );

    expect(screen.getByText("Disk encryption enabled")).toBeInTheDocument();
    expect(
      screen.getByText("Checks that FileVault is enabled.")
    ).toBeInTheDocument();
    expect(screen.getByText("Resolve:")).toBeInTheDocument();
    expect(
      screen.getByText("Enable FileVault in System Settings.")
    ).toBeInTheDocument();
    expect(screen.queryByText(/missing description/)).not.toBeInTheDocument();
  });

  it("renders only description or resolution without showing empty state", () => {
    const { unmount } = renderWithSetup(
      <PolicyDetailsModal
        onCancel={noop}
        policy={createMockPolicy({ description: "Some description" })}
      />
    );

    expect(screen.getByText("Some description")).toBeInTheDocument();
    expect(screen.queryByText("Resolve:")).not.toBeInTheDocument();
    expect(screen.queryByText(/missing description/)).not.toBeInTheDocument();

    unmount();

    renderWithSetup(
      <PolicyDetailsModal
        onCancel={noop}
        policy={createMockPolicy({ resolution: "Some resolution" })}
      />
    );

    expect(screen.getByText("Resolve:")).toBeInTheDocument();
    expect(screen.getByText("Some resolution")).toBeInTheDocument();
    expect(screen.queryByText(/missing description/)).not.toBeInTheDocument();
  });

  it("renders empty state when no description or resolution", () => {
    renderWithSetup(
      <PolicyDetailsModal onCancel={noop} policy={createMockPolicy()} />
    );

    expect(
      screen.getByText(
        "This policy is missing description and resolution instructions."
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("Resolve:")).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Please contact your IT admin/)
    ).not.toBeInTheDocument();
  });

  it("renders device user empty state with IT admin message", () => {
    renderWithSetup(
      <PolicyDetailsModal
        onCancel={noop}
        policy={createMockPolicy()}
        isDeviceUser
      />
    );

    expect(
      screen.getByText(
        /missing description and resolution instructions.*Please contact your IT admin/
      )
    ).toBeInTheDocument();
  });

  it("renders 'Resolve later' button only for failing conditional access policies", () => {
    const { unmount } = renderWithSetup(
      <PolicyDetailsModal
        onCancel={noop}
        policy={createMockPolicy({
          conditional_access_enabled: true,
          response: "fail",
          description: "Must comply",
        })}
        onResolveLater={jest.fn()}
      />
    );

    expect(screen.getByText("Resolve later")).toBeInTheDocument();

    unmount();

    renderWithSetup(
      <PolicyDetailsModal
        onCancel={noop}
        policy={createMockPolicy({
          conditional_access_enabled: true,
          response: "pass",
          description: "All good",
        })}
        onResolveLater={jest.fn()}
      />
    );

    expect(screen.queryByText("Resolve later")).not.toBeInTheDocument();
  });
});
