import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import { ISelectLabel, ISelectTeam } from "interfaces/target";
import TargetChipSelector from "./TargetChipSelector";

describe("TargetChipSelector", () => {
  const mockOnClick = jest.fn();

  const mockLabel: ISelectLabel = {
    id: 1,
    name: "Example Label",
    label_type: "regular",
    description: "A test label",
  };

  const mockTeam: ISelectTeam = {
    id: 2,
    name: "Example Team",
    description: "A test team",
  };

  it("renders the correct display text for a label", () => {
    render(
      <TargetChipSelector
        entity={mockLabel}
        isSelected={false}
        onClick={mockOnClick}
      />
    );

    expect(screen.getByText("Example Label")).toBeInTheDocument();
  });

  it("renders the correct display text for a team", () => {
    render(
      <TargetChipSelector
        entity={mockTeam}
        isSelected={false}
        onClick={mockOnClick}
      />
    );

    expect(screen.getByText("Example Team")).toBeInTheDocument();
  });

  it("renders the correct icon when selected", () => {
    render(
      <TargetChipSelector entity={mockLabel} isSelected onClick={mockOnClick} />
    );

    expect(screen.getByLabelText("check")).toBeInTheDocument();
  });

  it("renders the correct icon when not selected", () => {
    render(
      <TargetChipSelector
        entity={mockLabel}
        isSelected={false}
        onClick={mockOnClick}
      />
    );

    expect(screen.getByLabelText("plus")).toBeInTheDocument();
  });

  it("calls the onClick handler with the correct entity when clicked", () => {
    render(
      <TargetChipSelector
        entity={mockLabel}
        isSelected={false}
        onClick={(value) => (event) => mockOnClick(value, event)}
      />
    );

    fireEvent.click(screen.getByRole("button"));

    expect(mockOnClick).toHaveBeenCalledWith(mockLabel, expect.any(Object));
  });

  it("applies the correct data-selected attribute when selected", () => {
    render(
      <TargetChipSelector entity={mockLabel} isSelected onClick={mockOnClick} />
    );

    const button = screen.getByRole("button");
    expect(button).toHaveAttribute("data-selected", "true");
  });

  it("applies the correct data-selected attribute when not selected", () => {
    render(
      <TargetChipSelector
        entity={mockLabel}
        isSelected={false}
        onClick={mockOnClick}
      />
    );

    const button = screen.getByRole("button");
    expect(button).toHaveAttribute("data-selected", "false");
  });
});
