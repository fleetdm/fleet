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
  it("adds the icon-only class when a secondary button has only an icon", () => {
    const { container } = render(
      <Button variant="secondary">
        <Icon name="trash" />
      </Button>
    );
    expect(container.firstChild).toHaveClass("button--icon-only");
  });
  it("does not add the icon-only class when a secondary button has a text label", () => {
    const { container } = render(
      <Button variant="secondary">
        Delete <Icon name="trash" />
      </Button>
    );
    expect(container.firstChild).not.toHaveClass("button--icon-only");
  });
});
