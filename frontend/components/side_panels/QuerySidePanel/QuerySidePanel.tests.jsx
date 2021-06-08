import React from "react";
import { mount } from "enzyme";

import { stubbedOsqueryTable } from "test/helpers";

import QuerySidePanel from "./QuerySidePanel";

describe("QuerySidePanel - component", () => {
  const onOsqueryTableSelect = jest.fn();
  const onTextEditorInputChange = jest.fn();
  const props = {
    onOsqueryTableSelect,
    onTextEditorInputChange,
    selectedOsqueryTable: stubbedOsqueryTable,
  };

  it("renders the selected table in the dropdown", () => {
    const component = mount(<QuerySidePanel {...props} />);
    const tableSelect = component.find("Dropdown");

    expect(tableSelect.prop("value")).toEqual("users");
  });

  it("calls the onOsqueryTableSelect prop when a new table is selected in the dropdown", () => {
    const component = mount(<QuerySidePanel {...props} />);
    component.instance().onSelectTable("groups");

    expect(onOsqueryTableSelect).toHaveBeenCalledWith("groups");
  });
});
