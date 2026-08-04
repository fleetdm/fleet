import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import DropdownWrapper, { CustomOptionType } from "./DropdownWrapper";

const sampleOptions: CustomOptionType[] = [
  {
    label: "Option 1",
    value: "option1",
    tooltipContent: "Tooltip 1",
    helpText: "Help text 1",
  },
  {
    label: "Option 2",
    value: "option2",
    tooltipContent: "Tooltip 2",
    helpText: "Help text 2",
  },
];

describe("DropdownWrapper Component", () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders with help text", () => {
    render(
      <DropdownWrapper
        options={sampleOptions}
        value="option1"
        onChange={mockOnChange}
        name="test-dropdown"
        label="Test Dropdown"
        helpText="This is a help text."
      />
    );

    expect(screen.getByText(/test dropdown/i)).toBeInTheDocument();
    expect(screen.getByText(/this is a help text/i)).toBeInTheDocument();
  });

  test("calls onChange when an option is selected", async () => {
    render(
      <DropdownWrapper
        options={sampleOptions}
        value="option1"
        onChange={mockOnChange}
        name="test-dropdown"
        label="Test Dropdown"
        placeholder="Choose option"
      />
    );

    // Open the dropdown
    await userEvent.click(screen.getByText(/option 1/i));

    // Select Option 2
    await userEvent.click(screen.getByText(/option 2/i));

    expect(mockOnChange).toHaveBeenCalledWith({
      helpText: "Help text 2",
      label: "Option 2",
      tooltipContent: "Tooltip 2",
      value: "option2",
    });
  });

  test("renders error message when provided", () => {
    render(
      <DropdownWrapper
        options={sampleOptions}
        value="option1"
        onChange={mockOnChange}
        name="test-dropdown"
        label="Test Dropdown"
        error="This is an error message."
      />
    );

    expect(screen.getByText(/this is an error message/i)).toBeInTheDocument();
  });

  test("displays no options message when no options are available", async () => {
    render(
      <DropdownWrapper
        options={[]}
        value=""
        onChange={mockOnChange}
        name="test-dropdown"
        label="Test Dropdown"
        placeholder="Choose option"
      />
    );

    // Open dropdown
    await userEvent.click(screen.getByText(/choose option/i));

    expect(screen.getByText(/no results found/i)).toBeInTheDocument();
  });
});
