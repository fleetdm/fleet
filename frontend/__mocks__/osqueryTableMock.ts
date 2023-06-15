import { DEFAULT_OSQUERY_TABLE, IOsQueryTable } from "interfaces/osquery_table";

const createMockOsqueryTable = (
  overrides?: Partial<IOsQueryTable>
): IOsQueryTable => {
  return { ...DEFAULT_OSQUERY_TABLE, ...overrides };
};

export default createMockOsqueryTable;
