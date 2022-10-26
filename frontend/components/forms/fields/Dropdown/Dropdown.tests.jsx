import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import Dropdown from "components/forms/fields/Dropdown";

describe("Dropdown - component", () => {
  const options = [
    { text: "Users", value: "users" },
    { text: "Groups", value: "groups" },
  ];

  const props = {
    name: "my-select",
    options,
  };

  it("renders the dropdown", () => {
    render(<Dropdown {...props} />);
    const dropdownSelect = screen.getByRole("combobox");

    expect(dropdownSelect).toBeInTheDocument();
  });

  it("selects a value from dropdown", async () => {
    const onChangeSpy = jest.fn();
    const { user } = renderWithSetup(
      <Dropdown {...props} onChange={onChangeSpy} />
    );
    const inputNode = screen.getByRole("combobox");

    await user.type(inputNode, "users");
    await user.click(screen.getByRole("option"));

    expect(onChangeSpy).toHaveBeenCalledWith("users");
  });
});
