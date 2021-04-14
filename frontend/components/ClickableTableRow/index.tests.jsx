import React from "react";
import { mount } from "enzyme";

import ClickableTableRow from "./index";

const clickSpy = jest.fn();
const dblClickSpy = jest.fn();

const props = {
  onClick: clickSpy,
  onDoubleClick: dblClickSpy,
};

describe("ClickableTableRow - component", () => {
  it("calls onDblClick when row is double clicked", () => {
    const queryRow = mount(<ClickableTableRow {...props} />);
    queryRow.find("tr").simulate("doubleclick");
    expect(dblClickSpy).toHaveBeenCalled();
  });

  it("calls onSelect when row is clicked", () => {
    const queryRow = mount(<ClickableTableRow {...props} />);
    queryRow.find("tr").simulate("click");
    expect(clickSpy).toHaveBeenCalled();
  });
});
