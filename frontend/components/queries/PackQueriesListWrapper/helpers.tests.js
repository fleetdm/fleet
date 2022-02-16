import helpers from "./helpers";

describe("ScheduledQueriesListWrapper - helpers", () => {
  describe("#filterQueries", () => {
    const { filterQueries } = helpers;

    it("returns an empty array when given an empty array", () => {
      const queries = [];

      expect(filterQueries(queries, "text")).toEqual(queries);
    });

    it("returns an array of queries that have a matching query name", () => {
      const q1 = { name: "first query" };
      const q2 = { name: "second query" };
      const q3 = { name: "third query" };
      const queries = [q1, q2, q3];

      expect(filterQueries(queries, "")).toEqual(queries);
      expect(filterQueries(queries, "query")).toEqual(queries);
      expect(filterQueries(queries, "first")).toEqual([q1]);
      expect(filterQueries(queries, "second")).toEqual([q2]);
      expect(filterQueries(queries, "third")).toEqual([q3]);
    });
  });
});
