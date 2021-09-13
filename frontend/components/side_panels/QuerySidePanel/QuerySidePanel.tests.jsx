import React from "react";
import { mount } from "enzyme";

import { stubbedOsqueryTable } from "test/helpers";

import QuerySidePanel from "./QuerySidePanel";

describe("QuerySidePanel - component", () => {
  const onOsqueryTableSelect = jest.fn();
  const onClose = jest.fn();
  const props = {
    onOsqueryTableSelect,
    onClose,
    selectedOsqueryTable: stubbedOsqueryTable,
  };

  it("renders the selected table in the dropdown", () => {
    const component = mount(<QuerySidePanel {...props} />);
    const tableSelect = component.find("Dropdown");

    expect(tableSelect.prop("value")).toEqual("users");
  });

  // TODO: Functional components cannot test functions using Enzyme since
  // `instance()` is for class components - we should rethink only using Cypress
  // it("calls the onOsqueryTableSelect prop when a new table is selected in the dropdown", () => {
  //   const component = mount(<QuerySidePanel {...props} />);
  //   component.instance().onSelectTable("groups");

  //   expect(onOsqueryTableSelect).toHaveBeenCalledWith("groups");
  // });
});
