import React from "react";
import { Column } from "react-table";

import {
  expectTruncationTooltip,
  mockTruncatedTextLayout,
  renderWithSetup,
} from "test/test-utils";

import generateColumnConfigsFromRows from "./QueryResultsTableConfig";

type IResultRow = { host_display_name: string; host_id: number; name: string };

// react-table's `Column` union includes group columns, which carry no `Cell`,
// so the property isn't reachable through that type alone.
type IColumnWithCell = Column<IResultRow> & { Cell?: React.ElementType };

describe("QueryResultsTableConfig - Host column", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("shows the full host name in a tooltip when it is truncated", async () => {
    const longHostName = `DESKTOP-${"0123456789ABCDEF".repeat(8)}`;
    mockTruncatedTextLayout();
    const hostColumn = generateColumnConfigsFromRows<IResultRow>([
      { host_display_name: longHostName, host_id: 1, name: "osquery" },
    ]).find((column) => column.id === "Host") as IColumnWithCell;
    const Cell = hostColumn.Cell as React.ElementType;

    const { user } = renderWithSetup(
      <Cell
        cell={{ value: longHostName, column: { id: "Host" } }}
        row={{ original: { host_id: 1 } }}
      />
    );

    await expectTruncationTooltip(user, longHostName);
  });
});
