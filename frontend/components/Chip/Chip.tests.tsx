import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import Chip from "./Chip";

describe("Chip", () => {
  it("renders text without an icon when icon is omitted", () => {
    render(<Chip text="Fleet-maintained" />);

    expect(screen.getByText("Fleet-maintained")).toBeInTheDocument();
    // The Icon component renders an svg; there should be none when no icon prop.
    expect(document.querySelector("svg")).toBeNull();
  });

  it("renders a leading icon when icon prop is provided", () => {
    render(<Chip icon="user" text="Self-service" />);

    expect(screen.getByText("Self-service")).toBeInTheDocument();
    expect(document.querySelector("svg")).not.toBeNull();
  });

  it("renders a trailing icon when trailingIcon prop is provided", () => {
    render(
      <Chip icon="refresh" text="Auto install" trailingIcon="chevron-right" />
    );

    expect(screen.getByText("Auto install")).toBeInTheDocument();
    // Two icons (refresh + chevron-right) so two svgs.
    expect(document.querySelectorAll("svg").length).toBe(2);
  });

  it("renders as a button when onClick is provided", () => {
    render(<Chip text="Auto install" onClick={() => undefined} />);
    expect(
      screen.getByRole("button", { name: /auto install/i })
    ).toBeInTheDocument();
  });

  it("calls onClick when the chip is clicked", async () => {
    const handler = jest.fn();
    render(<Chip text="Auto install" onClick={handler} />);

    await userEvent.click(screen.getByText("Auto install"));
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("does not render a button when onClick is not provided", () => {
    render(<Chip text="Fleet-maintained" />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("wraps the chip in a TooltipWrapper when tooltip is provided", () => {
    const { container } = render(
      <Chip text="Self-service" tooltip="Available in self-service" />
    );

    // The chip text is still discoverable through the wrapper.
    expect(screen.getByText("Self-service")).toBeInTheDocument();
    // The TooltipWrapper element is present (content renders on hover, but the
    // wrapper class is the structural signal that tooltip wiring kicked in).
    expect(
      container.querySelector(".component__tooltip-wrapper")
    ).not.toBeNull();
  });

  it("does not render a TooltipWrapper when tooltip is omitted", () => {
    const { container } = render(<Chip text="Self-service" />);
    expect(container.querySelector(".component__tooltip-wrapper")).toBeNull();
  });
});
