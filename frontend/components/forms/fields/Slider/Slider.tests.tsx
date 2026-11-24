import { render, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import Slider from "./Slider";

describe("Slider Component", () => {
  const defaultProps = {
    onChange: jest.fn(),
    value: false,
    inactiveText: "Off",
    activeText: "On",
  };

  it("renders correctly with default props", () => {
    render(<Slider {...defaultProps} />);
    expect(screen.getByText("Off")).toBeInTheDocument();
    expect(screen.getByRole("switch")).toHaveClass("fleet-slider");
  });

  it("renders active state correctly", () => {
    render(<Slider {...defaultProps} value />);
    expect(screen.getByText("On")).toBeInTheDocument();
    expect(screen.getByRole("switch")).toHaveClass("fleet-slider--active");
  });

  it("calls onChange when clicked", () => {
    render(<Slider {...defaultProps} />);
    fireEvent.click(screen.getByRole("switch"));
    expect(defaultProps.onChange).toHaveBeenCalledTimes(1);
  });

  it("is not the form's submit button when Enter is pressed in a field", async () => {
    const user = userEvent.setup();
    const onSubmit = jest.fn((evt: React.FormEvent) => evt.preventDefault());
    render(
      <form onSubmit={onSubmit}>
        <input aria-label="Name" />
        <Slider {...defaultProps} />
        <button type="submit">Save</button>
      </form>
    );

    await user.type(screen.getByLabelText("Name"), "x{Enter}");

    expect(defaultProps.onChange).not.toHaveBeenCalled();
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });
});
