import React from "react";
import { mount } from "enzyme";

import { queryStub } from "test/stubs";
import QueriesListRow from "./QueriesListRow";

const checkSpy = jest.fn();
const clickSpy = jest.fn();
const dblClickSpy = jest.fn();

const props = {
  checked: false,
  onCheck: checkSpy,
  onSelect: clickSpy,
  onDoubleClick: dblClickSpy,
  query: queryStub,
  selected: false,
};

describe("QueriesListRow - component", () => {
  it("calls onDblClick when row is double clicked", () => {
    const queryRow = mount(<QueriesListRow {...props} />);
    queryRow.find("ClickableTableRow").simulate("doubleclick");
    expect(dblClickSpy).toHaveBeenCalledWith(queryStub);
  });

  it("calls onSelect when row is clicked", () => {
    const queryRow = mount(<QueriesListRow {...props} />);
    queryRow.find("ClickableTableRow").simulate("click");
    expect(clickSpy).toHaveBeenCalledWith(queryStub);
  });

  it("calls onCheck when row is checked", () => {
    const queryRow = mount(<QueriesListRow {...props} />);
    queryRow.find("Checkbox").find("input").simulate("change");
    expect(checkSpy).toHaveBeenCalledWith(!props.checked, queryStub.id);
  });
});
