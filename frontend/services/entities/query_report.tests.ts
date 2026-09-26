import sendRequest from "services";

import queryReportAPI, { LOAD_ALL_MAX_RESULTS } from "./query_report";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = sendRequest as jest.MockedFunction<typeof sendRequest>;

const row = (hostId: number) => ({
  host_id: hostId,
  host_name: `host-${hostId}`,
  last_fetched: "2024-01-01T00:00:00Z",
  columns: { model: "USB Mouse" },
});

const page = (hostIds: number[], hasNext: boolean) => ({
  query_id: 1,
  results: hostIds.map(row),
  report_clipped: false,
  count: 3,
  meta: { has_next_results: hasNext, has_previous_results: false },
});

describe("queryReportAPI.loadAll", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
  });

  it("fetches pages in order until has_next_results is false", async () => {
    mockSendRequest
      .mockResolvedValueOnce(page([1, 2], true))
      .mockResolvedValueOnce(page([3], false));

    const results = await queryReportAPI.loadAll({
      id: 1,
      sortBy: [{ key: "host_name", direction: "asc" }],
      query: "mouse",
    });

    expect(results.map((r) => r.host_id)).toEqual([1, 2, 3]);
    expect(mockSendRequest).toHaveBeenCalledTimes(2);
    expect(mockSendRequest.mock.calls[0][1]).toContain("page=0");
    expect(mockSendRequest.mock.calls[1][1]).toContain("page=1");
    // Every page carries the same sort and search so the pages line up.
    mockSendRequest.mock.calls.forEach(([, path]) => {
      expect(path).toContain("order_key=host_name");
      expect(path).toContain("query=mouse");
    });
  });

  it("rejects instead of returning a partial set when the page bound is hit", async () => {
    mockSendRequest.mockResolvedValue(page([1], true));

    await expect(queryReportAPI.loadAll({ id: 1, sortBy: [] })).rejects.toThrow(
      `more than ${LOAD_ALL_MAX_RESULTS} results`
    );
  });
});
