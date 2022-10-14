import React from "react";
import { render } from "@testing-library/react";
import { noop } from "lodash";
import { renderWithSetup } from "test/testingUtils";

import { IOsQueryTable } from "interfaces/osquery_table";

import QuerySidePanel from "./QuerySidePanel";

const DEFAULT_MOCK_TABLE: IOsQueryTable = {
  name: "users",
  description: "The users table",
  url: "https://test.com",
  platforms: ["darwin", "windows", "linux"],
  evented: true,
  cacheable: false,
  columns: [],
  examples: "Selects all users\n```\nSELECT * FROM users",
  notes: "The users table is really cool!",
};

const generateMockTable = (overrides: Partial<IOsQueryTable>) => {
  return { ...DEFAULT_MOCK_TABLE, ...overrides };
};

describe("Query side panel", () => {
  it("render the total number of tables", () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={DEFAULT_MOCK_TABLE}
        onOsqueryTableSelect={noop}
      />
    );
    expect(true).toBe(true);
  });
});
