import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { noop } from "lodash";

import { DropdownButton } from "./DropdownButton";

describe("DropdownButton - component", () => {
  it("calls the clicked item's onClick attribute", () => {
    const optionSpy = jest.fn();
    const dropdownOptions = [
      { label: "btn1", onClick: noop },
      { label: "btn2", onClick: optionSpy },
    ];

    render(
      <DropdownButton options={dropdownOptions}>New Button</DropdownButton>
    );

    fireEvent.click(screen.getByText("New Button"));

    fireEvent.click(screen.getByRole("button", { name: "btn2" }));
    expect(optionSpy).toHaveBeenCalled();
  });
});
