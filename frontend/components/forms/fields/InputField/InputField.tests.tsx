import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithSetup } from "test/test-utils";

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

    expect(screen.getByTestId("copy-icon")).toBeInTheDocument();
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

  test("calls onChange with { name, value } when parseTarget is true", async () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="my-field"
        parseTarget
      />
    );

    await userEvent.type(screen.getByPlaceholderText(/enter text/i), "A");

    expect(mockOnChange).toHaveBeenCalledTimes(1);
    expect(mockOnChange).toHaveBeenCalledWith({
      name: "my-field",
      value: "A",
    });
  });

  test("renders as read-only when readOnly prop is true", () => {
    render(
      <InputField
        value="read only value"
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        readOnly
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).toBeDisabled();
  });

  test("auto-focuses the input when autofocus is true", () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        autofocus
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).toHaveFocus();
  });

  test("sets autocomplete to 'new-password' when blockAutoComplete is true", () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        blockAutoComplete
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).toHaveAttribute(
      "autocomplete",
      "new-password"
    );
  });

  test("sets data-1p-ignore when ignore1password is true", () => {
    render(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        ignore1password
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).toHaveAttribute(
      "data-1p-ignore",
      "true"
    );
  });

  test("renders tooltip on label hover", async () => {
    const { user } = renderWithSetup(
      <InputField
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        tooltip="Helpful tooltip text"
      />
    );

    await user.hover(screen.getByText(/test input/i));
    await waitFor(() => {
      expect(screen.getByText("Helpful tooltip text")).toBeInTheDocument();
    });
  });

  test("sets step, min, and max attributes on number input", () => {
    render(
      <InputField
        value={5}
        onChange={mockOnChange}
        label="Number Input"
        placeholder="Enter number"
        name="test-number"
        type="number"
        step={0.5}
        min={0}
        max={100}
      />
    );

    const input = screen.getByPlaceholderText(/enter number/i);
    expect(input).toHaveAttribute("step", "0.5");
    expect(input).toHaveAttribute("min", "0");
    expect(input).toHaveAttribute("max", "100");
  });

  test("copies value to clipboard when copy button is clicked", async () => {
    const writeTextMock = jest.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: writeTextMock },
      writable: true,
      configurable: true,
    });

    render(
      <InputField
        value="Copy me"
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        enableCopy
      />
    );

    const copyButton = screen.getByTestId("copy-icon").closest("button");
    if (!copyButton) throw new Error("Expected copy button to exist");
    await userEvent.click(copyButton);

    expect(writeTextMock).toHaveBeenCalledWith("Copy me");
    await waitFor(() => {
      expect(screen.getByText("Copied!")).toBeInTheDocument();
    });
  });

  test("renders show-secret eye toggle with enableCopy and enableShowSecret on password field", async () => {
    render(
      <InputField
        value="s3cret"
        onChange={mockOnChange}
        label="Password"
        placeholder="Enter password"
        name="test-password"
        type="password"
        enableCopy
        enableShowSecret
      />
    );

    // The eye icon should be present
    const eyeIcon = screen.getByTestId("eye-icon");
    expect(eyeIcon).toBeInTheDocument();

    // Initially the input type should be password
    const input = screen.getByPlaceholderText(/enter password/i);
    expect(input).toHaveAttribute("type", "password");

    // Click the eye toggle to reveal the secret
    const eyeButton = eyeIcon.closest("button");
    if (!eyeButton) throw new Error("Expected eye button to exist");
    await userEvent.click(eyeButton);

    // After toggling, the input type should be text
    expect(input).toHaveAttribute("type", "text");

    // Click again to hide
    await userEvent.click(eyeButton);
    expect(input).toHaveAttribute("type", "password");
  });

  test("renders copy button in textarea mode when enableCopy is true", () => {
    render(
      <InputField
        value="Textarea content"
        onChange={mockOnChange}
        label="Test Textarea"
        placeholder="Enter text"
        name="test-textarea"
        type="textarea"
        enableCopy
      />
    );

    expect(screen.getByTestId("copy-icon")).toBeInTheDocument();
  });
});
