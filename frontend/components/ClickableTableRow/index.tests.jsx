import React from "react";
import { screen } from "@testing-library/react";

import { renderWithSetup } from "test/testingUtils";
import ClickableTableRow from "./index";

const clickSpy = jest.fn();
const dblClickSpy = jest.fn();

const props = {
  onClick: clickSpy,
  onDoubleClick: dblClickSpy,
};

describe("ClickableTableRow - component", () => {
  it("calls onDblClick when row is double clicked", async () => {
    const { user } = renderWithSetup(<ClickableTableRow {...props} />);
    await user.dblClick(screen.getByRole("row"));
    expect(dblClickSpy).toHaveBeenCalled();
  });

  it("calls onSelect when row is clicked", async () => {
    const { user } = renderWithSetup(<ClickableTableRow {...props} />);
    await user.click(screen.getByRole("row"));
    expect(clickSpy).toHaveBeenCalled();
  });
});
