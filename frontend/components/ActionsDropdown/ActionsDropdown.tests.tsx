import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import ActionsDropdown from "./ActionsDropdown";

const DROPDOWN_OPTIONS = [
  { disabled: false, label: "Edit", value: "edit-query" },
  { disabled: false, label: "Show query", value: "show-query" },
  { disabled: true, label: "Delete", value: "delete-query" },
];
const PLACEHOLDER = "Actions";
const ON_CHANGE = (value: string) => {
  console.log(value);
};

describe("Actions dropdown", () => {
  it("renders dropdown placeholder and options", async () => {
    const { user } = renderWithSetup(
      <ActionsDropdown
        options={DROPDOWN_OPTIONS} // Test
        placeholder={PLACEHOLDER}
        onChange={ON_CHANGE}
      />
    );

    await user.click(screen.getByText("Actions"));

    expect(screen.queryAllByText(/edit/i)[1]).toBeInTheDocument(); // Aria shows Edit twice since it's focused
    expect(screen.queryByText(/show query/i)).toBeInTheDocument();
    expect(screen.queryByText(/delete/i)).toBeInTheDocument();
  });

  it("renders dropdown as disabled when disabled prop is true", () => {
    renderWithSetup(
      <ActionsDropdown
        options={DROPDOWN_OPTIONS}
        placeholder={PLACEHOLDER}
        onChange={ON_CHANGE}
        disabled // Test
      />
    );
    expect(screen.getByRole("combobox")).toBeDisabled();
  });

  it("calls onChange with correct value when an option is selected", async () => {
    const mockOnChange = jest.fn();
    const { user } = renderWithSetup(
      <ActionsDropdown
        options={DROPDOWN_OPTIONS}
        placeholder={PLACEHOLDER}
        onChange={mockOnChange}
      />
    );

    await user.click(screen.getByText("Actions"));
    await user.click(screen.getByText("Edit"));

    expect(mockOnChange).toHaveBeenCalledWith("edit-query");
  });

  it("renders disabled option as non-selectable", async () => {
    const { user } = renderWithSetup(
      <ActionsDropdown
        options={DROPDOWN_OPTIONS}
        placeholder={PLACEHOLDER}
        onChange={ON_CHANGE}
      />
    );

    await user.click(screen.getByText("Actions"));
    const deleteOption = screen.getByText("Delete");

    expect(deleteOption).toHaveAttribute("aria-disabled", "true");
  });

  it("closes the dropdown when clicking outside", async () => {
    const { user } = renderWithSetup(
      <ActionsDropdown
        options={DROPDOWN_OPTIONS}
        placeholder={PLACEHOLDER}
        onChange={ON_CHANGE}
      />
    );

    await user.click(screen.getByText("Actions"));
    expect(screen.getByText("Edit")).toBeVisible();

    await user.click(document.body);
    expect(screen.queryByText(/edit/i)).not.toBeInTheDocument();
  });
});
