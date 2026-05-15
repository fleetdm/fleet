import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithSetup } from "test/test-utils";
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
    const { user } = renderWithSetup(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        tooltip="This is a tooltip."
      />
    );

    await user.hover(screen.getByText(/test input/i));
    await waitFor(() => {
      const tooltip = screen.getByText("This is a tooltip.");
      expect(tooltip).toBeInTheDocument();
    });
  });

  test("renders icon when iconSvg is provided", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        iconSvg="search"
      />
    );

    expect(screen.getByTestId("search-icon")).toBeInTheDocument();
  });

  test("does not render icon when iconSvg is not provided", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    expect(screen.queryByTestId("search-icon")).not.toBeInTheDocument();
  });

  test("renders as disabled when disabled prop is true", () => {
    render(
      <InputFieldWithIcon
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

  test("does not allow typing when disabled", async () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        disabled
      />
    );

    await userEvent.type(
      screen.getByPlaceholderText(/enter text/i),
      "some text"
    );

    expect(mockOnChange).not.toHaveBeenCalled();
  });

  test("autofocuses the input when autofocus is true", () => {
    render(
      <InputFieldWithIcon
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

  test("does not autofocus the input when autofocus is false", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).not.toHaveFocus();
  });

  test("calls onClick when the input is clicked", async () => {
    const mockOnClick = jest.fn();

    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        onClick={mockOnClick}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
      />
    );

    await userEvent.click(screen.getByPlaceholderText(/enter text/i));

    expect(mockOnClick).toHaveBeenCalledTimes(1);
  });

  test("sets data-1p-ignore attribute when ignore1Password is true", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        ignore1Password
      />
    );

    expect(screen.getByPlaceholderText(/enter text/i)).toHaveAttribute(
      "data-1p-ignore",
      "true"
    );
  });

  test("does not render clear button when value is empty", () => {
    render(
      <InputFieldWithIcon
        value=""
        onChange={mockOnChange}
        label="Test Input"
        placeholder="Enter text"
        name="test-input"
        clearButton
      />
    );

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  test("applies password type styling when type is password and value is present", () => {
    render(
      <InputFieldWithIcon
        value="secret"
        onChange={mockOnChange}
        label="Password"
        placeholder="Enter password"
        name="test-password"
        type="password"
      />
    );

    const input = screen.getByPlaceholderText(/enter password/i);
    expect(input).toHaveAttribute("type", "password");
    expect(input).toHaveClass("input-icon-field__input--password");
  });
});
