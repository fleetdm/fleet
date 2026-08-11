import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ILabelPolicy } from "interfaces/label";

import PolicyLabelModal from "./PolicyLabelModal";

// Render react-router's <Link> as a plain anchor so we can assert the href
// without a surrounding <Router>. This mirrors the real behavior we care about:
// labels render as real anchors (not buttons), which lets the browser open
// them in a new tab via middle-click or cmd/ctrl-click.
jest.mock("react-router", () => ({
  Link: ({ to, children }: { to: string; children: React.ReactNode }) => (
    <a href={to}>{children}</a>
  ),
}));

const INCLUDE_LABELS: ILabelPolicy[] = [
  { id: 1, name: "Engineering" },
  { id: 2, name: "Design" },
];
const EXCLUDE_LABELS: ILabelPolicy[] = [{ id: 3, name: "Servers" }];

describe("PolicyLabelModal", () => {
  it("renders only the include section, as plain text, when only include labels are provided and getLabelPath is absent", () => {
    render(
      <PolicyLabelModal
        includeLabels={INCLUDE_LABELS}
        includeScopeLabel="have any"
        onClose={jest.fn()}
      />
    );

    expect(screen.getByText(/Policy targets hosts that/)).toBeInTheDocument();
    expect(screen.getByText("have any")).toBeInTheDocument();
    expect(screen.getByText("Engineering")).toBeInTheDocument();
    expect(screen.getByText("Design")).toBeInTheDocument();

    // Without getLabelPath, labels render as plain text rather than links.
    expect(
      screen.queryByRole("link", { name: "Engineering" })
    ).not.toBeInTheDocument();

    expect(
      screen.queryByText(/Policy excludes hosts that/)
    ).not.toBeInTheDocument();
  });

  it("renders only the exclude section when only exclude labels are provided", () => {
    render(
      <PolicyLabelModal
        excludeLabels={EXCLUDE_LABELS}
        excludeScopeLabel="exclude all"
        onClose={jest.fn()}
      />
    );

    expect(screen.getByText(/Policy excludes hosts that/)).toBeInTheDocument();
    expect(screen.getByText("exclude all")).toBeInTheDocument();
    expect(screen.getByText("Servers")).toBeInTheDocument();

    expect(
      screen.queryByText(/Policy targets hosts that/)
    ).not.toBeInTheDocument();
  });

  it("renders both include and exclude sections together", () => {
    render(
      <PolicyLabelModal
        includeLabels={INCLUDE_LABELS}
        includeScopeLabel="have all"
        excludeLabels={EXCLUDE_LABELS}
        excludeScopeLabel="exclude any"
        onClose={jest.fn()}
      />
    );

    expect(screen.getByText(/Policy targets hosts that/)).toBeInTheDocument();
    expect(screen.getByText("have all")).toBeInTheDocument();
    expect(screen.getByText(/Policy excludes hosts that/)).toBeInTheDocument();
    expect(screen.getByText("exclude any")).toBeInTheDocument();
    expect(screen.getByText("Engineering")).toBeInTheDocument();
    expect(screen.getByText("Servers")).toBeInTheDocument();
  });

  it("renders no label sections when no labels are provided", () => {
    render(<PolicyLabelModal onClose={jest.fn()} />);

    expect(
      screen.queryByText(/Policy targets hosts that/)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Policy excludes hosts that/)
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  it("renders labels as anchor links to the path from getLabelPath when it is provided", () => {
    const getLabelPath = (labelId: number) => `/labels/${labelId}`;
    render(
      <PolicyLabelModal
        includeLabels={INCLUDE_LABELS}
        includeScopeLabel="have any"
        getLabelPath={getLabelPath}
        onClose={jest.fn()}
      />
    );

    // Real anchors (with href) are what make middle-click / cmd-click "open in
    // new tab" work — a <button> would not.
    const link = screen.getByRole("link", { name: "Design" });
    expect(link).toHaveAttribute("href", "/labels/2");
  });

  it("calls onClose when the Done button is clicked", async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    render(
      <PolicyLabelModal
        includeLabels={INCLUDE_LABELS}
        includeScopeLabel="have any"
        onClose={onClose}
      />
    );

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
