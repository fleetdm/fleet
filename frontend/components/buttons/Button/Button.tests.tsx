import React from "react";
import { render, fireEvent, screen } from "@testing-library/react";
import Icon from "components/Icon";
import Button from "./Button";

describe("Button component", () => {
  it("renders button with correct text", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText("Click me")).toBeInTheDocument();
  });
  it("applies correct class names", () => {
    const { container } = render(<Button disabled>Test</Button>);
    expect(container.firstChild).toHaveClass(
      "button button--default button--disabled"
    );
  });
  it("calls onClick when clicked", () => {
    const handleClick = jest.fn();
    render(<Button onClick={handleClick}>Click me</Button>);
    fireEvent.click(screen.getByText("Click me"));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
  it("calls onClick when Enter key is pressed", () => {
    const handleClick = jest.fn();
    render(<Button onClick={handleClick}>Press enter</Button>);
    fireEvent.keyDown(screen.getByText("Press enter"), { key: "Enter" });
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
  it("does not call onClick when disabled", () => {
    const handleClick = jest.fn();
    render(
      <Button onClick={handleClick} disabled>
        Disabled
      </Button>
    );
    fireEvent.click(screen.getByText("Disabled"));
    expect(handleClick).not.toHaveBeenCalled();
  });
  it("renders spinner when isLoading is true", () => {
    render(<Button isLoading>Loading</Button>);
    expect(screen.getByText("Loading")).toHaveClass("transparent-text");
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });
  it("applies custom className", () => {
    const { container } = render(
      <Button className="custom-class">Custom</Button>
    );
    expect(container.firstChild).toHaveClass("custom-class");
  });
  it("renders with correct title attribute", () => {
    render(<Button title="Button title">Titled button</Button>);
    expect(screen.getByTitle("Button title")).toBeInTheDocument();
  });
  it("applies the bordered secondary variant class", () => {
    const { container } = render(
      <Button variant="secondary">Secondary</Button>
    );
    expect(container.firstChild).toHaveClass("button button--secondary");
  });
  it("applies the subdued variant class", () => {
    const { container } = render(<Button variant="subdued">Subdued</Button>);
    expect(container.firstChild).toHaveClass("button button--subdued");
  });
  it("applies the small modifier for a small secondary button", () => {
    const { container } = render(
      <Button variant="secondary" size="small">
        Secondary
      </Button>
    );
    expect(container.firstChild).toHaveClass("button--secondary__small");
  });
  it("adds the icon-only class on a secondary button with icon and no label", () => {
    const { container } = render(
      <Button variant="secondary" icon="trash" ariaLabel="Delete" />
    );
    expect(container.firstChild).toHaveClass("button--icon-only");
  });
  it("sizes the icon to small on a small icon-only button", () => {
    render(
      <Button
        variant="secondary"
        size="small"
        icon="trash"
        ariaLabel="Delete"
      />
    );
    // Small icon = 12px per ICON_SIZES.
    expect(
      screen.getByTestId("trash-icon").querySelector("svg")
    ).toHaveAttribute("width", "12");
  });
  it("renders icon left of the label by default and applies the with-icon class", () => {
    const { container } = render(
      <Button variant="secondary" icon="plus">
        Add
      </Button>
    );
    expect(container.firstChild).toHaveClass("button--with-icon");
    expect(container.firstChild).not.toHaveClass("button--icon-only");
    expect(screen.getByTestId("plus-icon")).toBeInTheDocument();
  });
  it("renders icon right of the label when iconPosition is right", () => {
    const { container } = render(
      <Button variant="subdued" icon="chevron-right" iconPosition="right">
        Next
      </Button>
    );
    expect(container.firstChild).toHaveClass("button--with-icon");
    expect(container.firstChild).not.toHaveClass("button--icon-only");
    const wrapper = container.querySelector(".children-wrapper");
    expect(wrapper?.lastElementChild).toBe(
      screen.getByTestId("chevron-right-icon")
    );
  });
  it("does not render the icon or apply icon classes on variants outside the icon-enabled set", () => {
    const { container } = render(
      <Button variant="pill" icon="plus">
        Add
      </Button>
    );
    expect(container.firstChild).not.toHaveClass("button--with-icon");
    expect(container.firstChild).not.toHaveClass("button--icon-only");
    expect(screen.queryByTestId("plus-icon")).not.toBeInTheDocument();
  });
  it("auto-colors the icon white on white-text variants (default)", () => {
    render(
      <Button variant="default" icon="plus">
        Add
      </Button>
    );
    expect(
      screen
        .getByTestId("plus-icon")
        .querySelector("path")
        ?.getAttribute("stroke")
    ).toContain("core-fleet-white");
  });
  it("auto-colors the icon white on the alert variant", () => {
    render(
      <Button variant="alert" icon="plus">
        Delete
      </Button>
    );
    expect(
      screen
        .getByTestId("plus-icon")
        .querySelector("path")
        ?.getAttribute("stroke")
    ).toContain("core-fleet-white");
  });
  it("does not render the icon on the oversized variant (icons not enabled)", () => {
    render(
      <Button variant="oversized" icon="plus">
        Continue
      </Button>
    );
    expect(screen.queryByTestId("plus-icon")).not.toBeInTheDocument();
  });
  it("matches the icon color to the text on secondary/subdued", () => {
    render(
      <Button variant="secondary" icon="plus">
        Add
      </Button>
    );
    expect(
      screen
        .getByTestId("plus-icon")
        .querySelector("path")
        ?.getAttribute("stroke")
    ).toContain("ui-fleet-black-75");
  });
  it("treats a false child as no label and applies icon-only styling", () => {
    // Callers commonly do `{cond && "Label"}` — a false-y cond means no label.
    const { container } = render(
      <Button variant="secondary" icon="trash" ariaLabel="Delete">
        {false && "Delete"}
      </Button>
    );
    expect(container.firstChild).toHaveClass("button--icon-only");
    expect(container.firstChild).not.toHaveClass("button--with-icon");
  });
  it("renders aria-controls when ariaControls is provided", () => {
    render(<Button ariaControls="menu-1">Open menu</Button>);
    expect(screen.getByRole("button")).toHaveAttribute(
      "aria-controls",
      "menu-1"
    );
  });
  it("omits aria-controls when ariaControls is undefined", () => {
    render(<Button>Plain</Button>);
    expect(screen.getByRole("button")).not.toHaveAttribute("aria-controls");
  });
  it("warns in dev when an icon-only button has neither ariaLabel nor title", () => {
    const warn = jest
      .spyOn(console, "warn")
      .mockImplementation(() => undefined);
    render(<Button variant="secondary" icon="trash" />);
    expect(warn).toHaveBeenCalledWith(
      expect.stringContaining("no ariaLabel or title")
    );
    warn.mockRestore();
  });
  it("does not warn when an icon-only button has a title but no ariaLabel", () => {
    const warn = jest
      .spyOn(console, "warn")
      .mockImplementation(() => undefined);
    render(<Button variant="secondary" icon="trash" title="Delete" />);
    expect(warn).not.toHaveBeenCalled();
    warn.mockRestore();
  });
  it("adds the icon-only class for a legacy <Icon> child pattern on secondary/subdued", () => {
    // Some call sites keep the child <Icon> pattern to pass color/className.
    // We still want the square icon-only styling for those.
    const { container } = render(
      <Button variant="secondary" ariaLabel="Close">
        <Icon name="close" color="core-fleet-black" />
      </Button>
    );
    expect(container.firstChild).toHaveClass("button--icon-only");
  });
});
