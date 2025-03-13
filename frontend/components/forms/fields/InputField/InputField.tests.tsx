import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

// @ts-ignore
import InputField from "./InputField";

describe("InputField Component", () => {
  const mockOnChange = jest.fn();
  const mockOnBlur = jest.fn();
  const mockOnFocus = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders with label and placeholder", () => {
    render(
      <InputField
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
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    await userEvent.type(
      screen.getByPlaceholderText(/enter text/i),
      "New Value"
    );
    expect(mockOnChange).toHaveBeenCalledTimes(9); // 'New Value' has 9 characters
  });

  test("renders help text when provided", () => {
    render(
      <InputField
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
      <InputField
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

  test("renders as textarea when type is textarea", () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Textarea"
        placeholder="Enter text"
        name="test-textarea"
        type="textarea"
      />
    );

    expect(screen.getByRole("textbox")).toHaveAttribute(
      "name",
      "test-textarea"
    );
  });

  test("renders copy button when enableCopy is true", () => {
    render(
      <InputField
        value="Some Value"
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        enableCopy
      />
    );

    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
  });

  test("calls onBlur when input loses focus", async () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        onBlur={mockOnBlur}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    const input = screen.getByPlaceholderText(/enter text/i);
    await userEvent.click(input);
    await userEvent.tab();

    expect(mockOnBlur).toHaveBeenCalledTimes(1);
  });

  test("calls onFocus when input gains focus", async () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        onFocus={mockOnFocus}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    const input = screen.getByPlaceholderText(/enter text/i);
    await userEvent.click(input);

    expect(mockOnFocus).toHaveBeenCalledTimes(1);
  });

  test("renders as disabled when disabled prop is true", () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        disabled
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).toBeDisabled();
  });
});
