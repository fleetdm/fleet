import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import DropdownCell from "./DropdownCell";

const DROPDOWN_OPTIONS = [
  { disabled: false, label: "Edit", value: "edit-query" },
  { disabled: false, label: "Show query", value: "show-query" },
  { disabled: true, label: "Delete", value: "delete-query" },
];
const PLACEHOLDER = "Actions";
const ON_CHANGE = (value: string) => {
  console.log(value);
};

describe("Dropdown cell", () => {
  it("renders dropdown placeholder and options", async () => {
    const { user } = renderWithSetup(
      <DropdownCell
        options={DROPDOWN_OPTIONS}
        placeholder={PLACEHOLDER}
        onChange={ON_CHANGE}
      />
    );

    await user.click(screen.getByText("Actions"));

    expect(screen.getByText(/edit/i)).toBeInTheDocument();
    expect(screen.getByText(/show query/i)).toBeInTheDocument();
    expect(screen.getByText(/delete/i)).toBeInTheDocument();
  });
});
