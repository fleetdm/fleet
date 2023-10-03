import { IQueryReport } from "interfaces/query_report";

const DEFAULT_QUERY_REPORT_MOCK: IQueryReport = {
  query_id: 31,
  results: [
    {
      host_id: 1,
      host_name: "foo",
      last_fetched: "2021-01-19T17:08:31Z",
      columns: {
        model: "USB 2.0 Hub",
        vendor: "VIA Labs, Inc.",
      },
    },
    {
      host_id: 1,
      host_name: "foo",
      last_fetched: "2021-01-19T17:08:31Z",
      columns: {
        model: "USB Keyboard",
        vendor: "VIA Labs, Inc.",
      },
    },
    {
      host_id: 2,
      host_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB Reciever",
        vendor: "Logitech",
      },
    },
    {
      host_id: 2,
      host_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "USB Reciever",
        vendor: "Logitech",
      },
    },
    {
      host_id: 2,
      host_name: "bar",
      last_fetched: "2021-01-19T17:20:00Z",
      columns: {
        model: "Display Audio",
        vendor: "Apple Inc.",
      },
    },
  ],
};

const createMockQueryReport = (
  overrides?: Partial<IQueryReport>
): IQueryReport => {
  return { ...DEFAULT_QUERY_REPORT_MOCK, ...overrides };
};

export default createMockQueryReport;
