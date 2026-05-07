import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { noop } from "lodash";
import { IHostPolicy } from "interfaces/policy";

import HostPolicies from "./HostPolicies";

const createMockHostPolicy = (
  overrides?: Partial<IHostPolicy>
): IHostPolicy => ({
  id: 1,
  name: "Test Policy",
  query: "SELECT 1;",
  description: "A test policy",
  author_id: 1,
  author_name: "Test User",
  author_email: "test@test.com",
  resolution: "Fix it",
  platform: "darwin",
  team_id: null,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  response: "pass",
  ...overrides,
});

const baseProps = {
  policies: [] as IHostPolicy[],
  isLoading: false,
  togglePolicyDetailsModal: noop,
  hostPlatform: "darwin",
};

const renderWithContext = (props = {}) =>
  createCustomRenderer()(<HostPolicies {...baseProps} {...props} />);

describe("HostPolicies", () => {
  it("renders empty state with Manage policies CTA when user has permission", () => {
    renderWithContext({
      canManagePolicies: true,
      onManagePolicies: jest.fn(),
    });

    expect(screen.getByText("No policies checked")).toBeInTheDocument();
    expect(
      screen.getByText(
        /Select Refetch to load the latest data from this host, or manage its policies/
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /manage policies/i })
    ).toBeInTheDocument();
  });

  it("renders empty state without Manage policies CTA or manage clause when user lacks permission", () => {
    renderWithContext({
      canManagePolicies: false,
    });

    expect(screen.getByText("No policies checked")).toBeInTheDocument();
    expect(
      screen.getByText(/Select Refetch to load the latest data from this host\./)
    ).toBeInTheDocument();
    expect(screen.queryByText(/manage its policies/)).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /manage policies/i })
    ).not.toBeInTheDocument();
  });

  it("renders device user copy when deviceUser is true", () => {
    renderWithContext({
      deviceUser: true,
      canManagePolicies: false,
    });

    expect(
      screen.getByText(/Select Refetch to load the latest data from your device\./)
    ).toBeInTheDocument();
  });

  it("renders policy count when policies exist", () => {
    const policies = [
      createMockHostPolicy({ id: 1, name: "Policy A", response: "pass" }),
      createMockHostPolicy({ id: 2, name: "Policy B", response: "fail" }),
      createMockHostPolicy({ id: 3, name: "Policy C", response: "pass" }),
    ];

    renderWithContext({ policies });

    expect(screen.getByText("3 policies")).toBeInTheDocument();
  });

  it("does not render for iOS hosts", () => {
    renderWithContext({ hostPlatform: "ios" });

    expect(
      screen.getByText(/policies are not supported for this host/i)
    ).toBeInTheDocument();
    expect(screen.queryByText("No policies checked")).not.toBeInTheDocument();
  });

  it("does not render for Android hosts", () => {
    renderWithContext({ hostPlatform: "android" });

    expect(
      screen.getByText(/policies are not supported for this host/i)
    ).toBeInTheDocument();
    expect(screen.queryByText("No policies checked")).not.toBeInTheDocument();
  });
});
