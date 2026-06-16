import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ISoftwareInstallPolicyUI } from "interfaces/software";

import PoliciesModal from "./PoliciesModal";

const POLICIES: ISoftwareInstallPolicyUI[] = [
  { id: 1, name: "Okta - Engineering", type: new Set(["dynamic"]) },
  { id: 2, name: "Okta - QA", type: new Set(["dynamic", "patch"]) },
];

describe("PoliciesModal", () => {
  it("renders the policies table with linked policies", () => {
    render(<PoliciesModal policies={POLICIES} teamId={3} onExit={jest.fn()} />);

    expect(screen.getAllByText("Okta - Engineering").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Okta - QA").length).toBeGreaterThan(0);
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
});
