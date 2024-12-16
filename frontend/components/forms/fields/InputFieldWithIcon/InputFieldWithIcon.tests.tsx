import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
// @ts-ignore
import InputFieldWithIcon from "./InputFieldWithIcon";

describe("InputFieldWithIcon Component", () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders with label and placeholder", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    expect(screen.getByText(/test input/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/enter text/i)).toBeInTheDocument();
  });

  test("calls onChange when input value changes", async () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    // Change the input value
    await userEvent.type(
      screen.getByPlaceholderText(/enter text/i),
      "New Value"
    );

    expect(mockOnChange).toHaveBeenCalledTimes(9); // 'New Value' has 9 characters
  });

  test("renders help text when provided", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        helpText="This is a help text."
      />
    );

    expect(screen.getByText(/this is a help text/i)).toBeInTheDocument();
  });

  test("renders error message when provided", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        error="This is an error message."
      />
    );

    expect(screen.getByText(/this is an error message/i)).toBeInTheDocument();
  });

  test("renders clear button when clearButton is true and input has value", () => {
    render(
      <InputFieldWithIcon
        value="Some Value"
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        clearButton
      />
    );

    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  test("clears input value when clear button is clicked", async () => {
    render(
      <InputFieldWithIcon
        value="Some Value"
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        clearButton
      />
    );

    // Click the clear button
    await userEvent.click(screen.getByRole("button"));

    expect(mockOnChange).toHaveBeenCalledTimes(1);
    expect(mockOnChange).toHaveBeenCalledWith("");
  });

  test("renders tooltip when provided", async () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        tooltip="This is a tooltip."
      />
    );

    await fireEvent.mouseEnter(screen.getByText(/test input/i));
    const tooltip = screen.getByText("This is a tooltip.");
    expect(tooltip).toBeInTheDocument();
  });
});
