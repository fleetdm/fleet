import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ISoftwareInstallPolicyUI } from "interfaces/software";

import PoliciesModal from "./PoliciesModal";

// Spy on the table so we can assert teamId flows through; the link's href
// itself can't be asserted in tests because react-router 3's <Link> needs a
// router context to resolve `to` into an href.
const mockInstallerPoliciesTable = jest.fn();
jest.mock("../SoftwareInstallerCard/InstallerPoliciesTable", () => ({
  __esModule: true,
  default: (props: { teamId?: number }) => {
    mockInstallerPoliciesTable(props);
    return <div data-testid="installer-policies-table-mock" />;
  },
}));

const POLICIES: ISoftwareInstallPolicyUI[] = [
  { id: 1, name: "Okta - Engineering", type: new Set(["dynamic"]) },
  { id: 2, name: "Okta - QA", type: new Set(["dynamic", "patch"]) },
];

beforeEach(() => {
  mockInstallerPoliciesTable.mockClear();
});

describe("PoliciesModal", () => {
  it("renders the policies table with linked policies", () => {
    render(<PoliciesModal policies={POLICIES} teamId={3} onExit={jest.fn()} />);

    expect(mockInstallerPoliciesTable).toHaveBeenCalledWith(
      expect.objectContaining({ policies: POLICIES })
    );
  });

  it("renders the empty state when no policies are linked", () => {
    render(<PoliciesModal policies={[]} onExit={jest.fn()} />);

    expect(
      screen.getByText("No policies are linked to this software.")
    ).toBeInTheDocument();
  });

  it("calls onExit when Done is clicked", async () => {
    const onExit = jest.fn();
    render(<PoliciesModal policies={POLICIES} onExit={onExit} />);

    await userEvent.click(screen.getByRole("button", { name: /done/i }));
    expect(onExit).toHaveBeenCalled();
  });

  it("forwards teamId to InstallerPoliciesTable", () => {
    // teamId is what the table threads into each policy LinkCell as fleet_id;
    // if the prop is dropped, every row's "View policy" link silently loses
    // fleet context.
    render(<PoliciesModal policies={POLICIES} teamId={7} onExit={jest.fn()} />);

    expect(mockInstallerPoliciesTable).toHaveBeenCalledWith(
      expect.objectContaining({ teamId: 7 })
    );
  });
});
