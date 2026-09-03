import { astify } from ".";

/**
 * Accept/reject cases live in corpus.json, driven by corpus.tests.ts. This file
 * is for assertions the corpus format cannot express.
 */
describe("osquery_sql_parser", () => {
  it("reports syntax error locations in the original text's coordinates", () => {
    // Both assertions live in the catch block, so if astify returns without
    // throwing, the assertion count fails the test.
    expect.assertions(2);
    try {
      // Leading blank lines must count toward the reported line number so
      // the location matches what the user sees in the editor.
      astify("\n\nSELECT *\nFRM users");
    } catch (err) {
      const { location } = err as {
        location?: { start: { line: number; column: number } };
      };
      expect(location?.start.line).toEqual(4);
      expect(location?.start.column).toEqual(1);
    }
  });
});
