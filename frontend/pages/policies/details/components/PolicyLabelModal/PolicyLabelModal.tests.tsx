import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ILabelPolicy } from "interfaces/label";

import PolicyLabelModal from "./PolicyLabelModal";

const INCLUDE_LABELS: ILabelPolicy[] = [
  { id: 1, name: "Engineering" },
  { id: 2, name: "Design" },
];
const EXCLUDE_LABELS: ILabelPolicy[] = [{ id: 3, name: "Servers" }];

describe("PolicyLabelModal", () => {
  it("renders only the include section, as plain text, when only include labels are provided and onLabelClick is absent", () => {
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

    // Without onLabelClick, labels render as plain text rather than buttons.
    expect(
      screen.queryByRole("button", { name: "Engineering" })
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

  it("renders labels as clickable buttons and calls onLabelClick with the label id when onLabelClick is provided", async () => {
    const user = userEvent.setup();
    const onLabelClick = jest.fn();
    render(
      <PolicyLabelModal
        includeLabels={INCLUDE_LABELS}
        includeScopeLabel="have any"
        onLabelClick={onLabelClick}
        onClose={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: "Design" }));

    expect(onLabelClick).toHaveBeenCalledWith(2);
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
