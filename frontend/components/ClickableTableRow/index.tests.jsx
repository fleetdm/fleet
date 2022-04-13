import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ClickableTableRow from "./index";

const clickSpy = jest.fn();
const dblClickSpy = jest.fn();

const props = {
  onClick: clickSpy,
  onDoubleClick: dblClickSpy,
};

describe("ClickableTableRow - component", () => {
  it("calls onDblClick when row is double clicked", () => {
    render(<ClickableTableRow {...props} />);
    userEvent.dblClick(screen.getByRole("row"));
    expect(dblClickSpy).toHaveBeenCalled();
  });

  it("calls onSelect when row is clicked", () => {
    render(<ClickableTableRow {...props} />);
    userEvent.click(screen.getByRole("row"));
    expect(clickSpy).toHaveBeenCalled();
  });
});
