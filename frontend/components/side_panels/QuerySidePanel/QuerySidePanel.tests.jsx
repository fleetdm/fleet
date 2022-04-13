import React from "react";
import { render, screen } from "@testing-library/react";

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
    render(<QuerySidePanel {...props} />);

    expect(screen.getByRole("combobox")).toBeInTheDocument();
    expect(screen.getByText("users")).toBeInTheDocument();
  });

  // TODO: Functional components cannot test functions using Enzyme since
  // `instance()` is for class components - we should rethink only using Cypress
  // it("calls the onOsqueryTableSelect prop when a new table is selected in the dropdown", () => {
  //   const component = mount(<QuerySidePanel {...props} />);
  //   component.instance().onSelectTable("groups");

  //   expect(onOsqueryTableSelect).toHaveBeenCalledWith("groups");
  // });
});
