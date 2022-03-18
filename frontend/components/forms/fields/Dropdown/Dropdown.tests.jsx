import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

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
    render(<Dropdown {...props} onChange={onChangeSpy} />);
    const inputNode = screen.getByRole("combobox");

    userEvent.type(inputNode, "users");
    fireEvent.mouseDown(screen.getByRole("option"));

    expect(onChangeSpy).toHaveBeenCalledWith("users");
  });
});
