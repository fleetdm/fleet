import { buildQueryStringFromParams } from ".";

describe("url utilites", () => {
  it("creates a query string from a params object", () => {
    const params = {
      query: "test",
      page: 1,
      order: "asc",
      isNew: true,
    };
    expect(buildQueryStringFromParams(params)).toBe(
      "query=test&page=1&order=asc&isNew=true"
    );
  });

  it("filters out undefined values", () => {
    const params = {
      query: undefined,
      page: 1,
      order: "asc",
    };
    expect(buildQueryStringFromParams(params)).toBe("page=1&order=asc");
  });

  it("filters out empty string values", () => {
    const params = {
      query: "",
      page: 1,
      order: "asc",
    };
    expect(buildQueryStringFromParams(params)).toBe("page=1&order=asc");
  });

  it("filters out null values", () => {
    const params = {
      query: null,
      page: 1,
      order: "asc",
    };
    expect(buildQueryStringFromParams(params)).toBe("page=1&order=asc");
  });
});
