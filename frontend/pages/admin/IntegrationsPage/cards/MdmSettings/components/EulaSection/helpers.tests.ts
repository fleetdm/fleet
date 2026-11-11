import { getErrorMessage, hasExpectedExtension } from "./helpers";

const apiError = (status: number, reason: string) => ({
  status,
  data: { message: "Bad request", errors: [{ name: "base", reason }] },
});

describe("EulaSection helpers", () => {
  it.each([
    ["darwin", "eula.pdf", true],
    ["darwin", "EULA.PDF", true],
    ["darwin", "terms.md", false],
    ["windows", "terms.md", true],
    ["windows", "terms.MD", true],
    ["windows", "terms.md.pdf", false],
  ] as const)("%s accepts %s: %s", (platform, fileName, expected) => {
    expect(hasExpectedExtension(platform, fileName)).toBe(expected);
  });

  it("shows the server's reason for a rejected upload", () => {
    expect(
      getErrorMessage(apiError(400, "The file has no text to show."))
    ).toBe("Couldn’t upload EULA. The file has no text to show.");
  });

  it.each([
    ["a server error", apiError(500, "inserting EULA: database is down")],
    ["a rejection without a reason", apiError(400, "")],
    ["a network error", "Network Error"],
  ])("falls back to a generic message for %s", (_, err) => {
    expect(getErrorMessage(err)).toBe(
      "Couldn’t upload EULA. Please try again."
    );
  });
});
