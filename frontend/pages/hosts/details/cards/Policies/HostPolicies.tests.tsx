import React, { useCallback, useState } from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, renderWithSetup } from "test/test-utils";
import { noop } from "lodash";
import { IHostPolicy } from "interfaces/policy";

import HostPolicies from "./HostPolicies";
import PolicyDetailsModal from "./HostPoliciesTable/PolicyDetailsModal";

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
  closePolicyDetailsModal: noop,
  hostPlatform: "darwin",
};

const renderWithContext = (props = {}) =>
  createCustomRenderer()(<HostPolicies {...baseProps} {...props} />);

describe("HostPolicies", () => {
  it("renders empty state with 0 count and Manage policies CTA when user has permission", () => {
    renderWithContext({
      canManagePolicies: true,
      onManagePolicies: jest.fn(),
    });

    expect(screen.getByText("0 policies")).toBeInTheDocument();
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
      screen.getByText(
        /Select Refetch to load the latest data from this host\./
      )
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
      screen.getByText(
        /Select Refetch to load the latest data from your device\./
      )
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

  it("keeps the current pagination page when a policy is selected after first page", async () => {
    const policies = Array.from({ length: 25 }, (_, i) =>
      createMockHostPolicy({ id: i + 1, name: `Policy ${i + 1}` })
    );

    const Wrapper = () => {
      const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
        null
      );

      const openModal = useCallback((p: IHostPolicy) => {
        setSelectedPolicy(p);
      }, []);

      const closeModal = useCallback(() => {
        setSelectedPolicy(null);
      }, []);

      return (
        <>
          <HostPolicies
            {...baseProps}
            policies={policies}
            togglePolicyDetailsModal={openModal}
            closePolicyDetailsModal={closeModal}
          />
          {selectedPolicy && (
            <PolicyDetailsModal onCancel={closeModal} policy={selectedPolicy} />
          )}
        </>
      );
    };

    const { user } = renderWithSetup(<Wrapper />);

    // Each policy name renders twice (visible cell + truncation tooltip).
    // Page 1 shows the first 20 policies.
    expect(screen.queryAllByText("Policy 1").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("Policy 21")).toHaveLength(0);

    // Go to page 2.
    await user.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.queryAllByText("Policy 1")).toHaveLength(0);
    expect(screen.queryAllByText("Policy 21").length).toBeGreaterThan(0);

    // Select a policy on page 2 (opens the details modal). Click the row
    // rather than the name cell, which is a router Link.
    const policyRow = screen.getAllByText("Policy 25")[0].closest("tr");
    expect(policyRow).not.toBeNull();
    await user.click(policyRow as HTMLElement);

    // Confirm the click actually selected the policy and opened the modal —
    // otherwise the page-retention assertions below could pass trivially.
    // "Resolve:" only renders inside PolicyDetailsModal.
    expect(screen.getByText("Resolve:")).toBeInTheDocument();

    // The table should still be on page 2 — Policy 21 belongs to page 2 and
    // is not rendered by the modal.
    expect(screen.queryAllByText("Policy 21").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("Policy 1")).toHaveLength(0);
  });

  it("closes the policy details modal when HostPolicies unmounts", async () => {
    // Simulates the policies tab unmounting (e.g. the user presses the
    // browser back button from /hosts/:id/policies to /hosts/:id, or
    // switches to a different host details tab). The modal state lives
    // in the parent (HostDetailsPage / DeviceUserPage), so the parent owns it here too.
    const policy = createMockHostPolicy({ name: "Failing policy" });

    const Wrapper = ({ showPolicies }: { showPolicies: boolean }) => {
      const [isModalOpen, setIsModalOpen] = useState(false);
      const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
        null
      );

      const openModal = useCallback((p: IHostPolicy) => {
        setSelectedPolicy(p);
        setIsModalOpen(true);
      }, []);

      const closeModal = useCallback(() => {
        setSelectedPolicy(null);
        setIsModalOpen(false);
      }, []);

      return (
        <>
          <button type="button" onClick={() => openModal(policy)}>
            open modal
          </button>
          {showPolicies && (
            <HostPolicies
              {...baseProps}
              policies={[policy]}
              togglePolicyDetailsModal={openModal}
              closePolicyDetailsModal={closeModal}
            />
          )}
          {isModalOpen && selectedPolicy && (
            <PolicyDetailsModal onCancel={closeModal} policy={selectedPolicy} />
          )}
        </>
      );
    };

    const { user, rerender } = renderWithSetup(<Wrapper showPolicies />);

    await user.click(screen.getByRole("button", { name: /open modal/i }));
    // "Resolve:" only renders inside PolicyDetailsModal.
    expect(screen.getByText("Resolve:")).toBeInTheDocument();
    rerender(<Wrapper showPolicies={false} />);
    expect(screen.queryByText("Resolve:")).not.toBeInTheDocument();
  });
});
