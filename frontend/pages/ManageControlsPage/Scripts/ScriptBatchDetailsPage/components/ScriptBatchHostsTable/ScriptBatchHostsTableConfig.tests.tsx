import { screen } from "@testing-library/react";
import React from "react";
import { Column } from "react-table";

import { createMockScriptBatchHostResults } from "__mocks__/scriptMock";
import { IScriptBatchHostResult } from "services/entities/scripts";
import { renderWithSetup } from "test/test-utils";

import generateColumnConfigs from "./ScriptBatchHostsTableConfig";

// react-table's `Column` union includes group columns, which carry no `Cell`,
// so the property isn't reachable through that type alone.
type IColumnWithCell = Column<IScriptBatchHostResult> & {
  Cell?: React.ElementType;
};

describe("ScriptBatchHostsTableConfig - Host name column", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("shows the full host name in a tooltip when it is truncated", async () => {
    const longHostName = `DESKTOP-${"0123456789ABCDEF".repeat(8)}`;
    // jsdom has no layout, so report a text box narrower than its content.
    jest
      .spyOn(HTMLElement.prototype, "offsetWidth", "get")
      .mockReturnValue(352);
    jest.spyOn(Element.prototype, "scrollWidth", "get").mockReturnValue(1000);
    const hostColumn = generateColumnConfigs("incompatible").find(
      (column) => column.accessor === "display_name"
    ) as IColumnWithCell;
    const Cell = hostColumn.Cell as React.ElementType;

    const host = {
      ...createMockScriptBatchHostResults("incompatible").hosts[0],
      display_name: longHostName,
    };

    const { user } = renderWithSetup(<Cell row={{ original: host }} />);

    await user.hover(
      screen.getByText(longHostName, {
        selector: ".data-table__tooltip-truncated-text",
      })
    );
    expect(
      await screen.findByText(longHostName, {
        selector: ".truncated-tooltip.show",
      })
    ).toBeVisible();
  });
});
