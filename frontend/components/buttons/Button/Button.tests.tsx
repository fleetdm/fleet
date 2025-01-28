import React from "react";
import { render, fireEvent, screen } from "@testing-library/react";
import Button from "./Button";

describe("Button component", () => {
  it("renders button with correct text", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText("Click me")).toBeInTheDocument();
  });
  it("applies correct class names", () => {
    const { container } = render(
      <Button variant="brand" size="large">
        Test
      </Button>
    );
    expect(container.firstChild).toHaveClass(
      "button button--brand button--large"
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
});
