import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import createMockOsqueryTable from "__mocks__/osqueryTableMock";
import QuerySidePanel from "./QuerySidePanel";

describe("QuerySidePanel - component", () => {
  it("renders the query side panel with the correct table selected", () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={(tableName: string) => noop}
        onClose={noop}
      />
    );

    const tableDropdownText = screen.getByText("Users");
    expect(tableDropdownText).toBeInTheDocument();
  });

  // it("renders platform compatibility", () => {
  //   render(
  //     <QuerySidePanel
  //       selectedOsqueryTable={createMockOsqueryTable()}
  //       onOsqueryTableSelect={(tableName: string) => noop}
  //       onClose={noop}
  //     />
  //   );

  //   const tableDropdownText = screen.getByText("Users");
  //   expect(tableDropdownText).toBeInTheDocument();
  // });

  // it("renders the correct number of columns", () => {
  //   render(
  //     <QuerySidePanel
  //       selectedOsqueryTable={createMockOsqueryTable()}
  //       onOsqueryTableSelect={(tableName: string) => noop}
  //       onClose={noop}
  //     />
  //   );

  //   const tableDropdownText = screen.getByText("Users");
  //   expect(tableDropdownText).toBeInTheDocument();
  // });
  // it("renders the correct column tooltip", () => {
  //   render(
  //     <QuerySidePanel
  //       selectedOsqueryTable={createMockOsqueryTable()}
  //       onOsqueryTableSelect={(tableName: string) => noop}
  //       onClose={noop}
  //     />
  //   );

  //   const tableDropdownText = screen.getByText("Users");
  //   expect(tableDropdownText).toBeInTheDocument();
  // });
  // it("render an example", () => {
  //   render(
  //     <QuerySidePanel
  //       selectedOsqueryTable={createMockOsqueryTable()}
  //       onOsqueryTableSelect={(tableName: string) => noop}
  //       onClose={noop}
  //     />
  //   );

  //   const tableDropdownText = screen.getByText("Users");
  //   expect(tableDropdownText).toBeInTheDocument();
  // });
  // it("render notes", () => {
  //   render(
  //     <QuerySidePanel
  //       selectedOsqueryTable={createMockOsqueryTable()}
  //       onOsqueryTableSelect={(tableName: string) => noop}
  //       onClose={noop}
  //     />
  //   );

  //   const tableDropdownText = screen.getByText("Users");
  //   expect(tableDropdownText).toBeInTheDocument();
  // });
  // it("renders a link to the source", () => {
  //   render(
  //     <QuerySidePanel
  //       selectedOsqueryTable={createMockOsqueryTable()}
  //       onOsqueryTableSelect={(tableName: string) => noop}
  //       onClose={noop}
  //     />
  //   );

  //   const text = screen.getByText("Source");
  //   const icon = screen.queryByTestId("Icon");

  //   expect(text).toBeInTheDocument();
  //   expect(icon).toBeNull();
  //   expect(text.closest("a")).toHaveAttribute(
  //     "href",
  //     "https://fleetdm.com/tables/users"
  //   );
  //   expect(text.closest("a")).not.toHaveAttribute("target", "_blank");
  // });
});
