import { validate } from "./WebAddress";

describe("WebAddress validate", () => {
  it("accepts a valid URL with surrounding whitespace", () => {
    expect(validate({ serverURL: " https://fleet.example.com " })).toEqual({});
  });

  it("still rejects an invalid URL", () => {
    expect(validate({ serverURL: " fleet example " }).serverURL).toBeDefined();
  });
});
