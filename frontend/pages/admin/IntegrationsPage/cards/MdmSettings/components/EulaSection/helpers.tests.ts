import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";

import { getErrorMessage, hasExpectedExtension } from "./helpers";

const apiError = (reason: string) =>
  ({
    data: { message: "Bad request", errors: [{ name: "base", reason }] },
  } as AxiosResponse<IApiError>);

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

  it.each([
    "The file is empty.",
    "The file must be UTF-8 text.",
    "The file contains HTML. Convert it to markdown and upload again.",
    "The file has no text to show.",
    "The file must be 512 KB or smaller.",
    "The file has a line longer than 8 KB. Split long paragraphs into shorter lines and upload again.",
    "The file has more than 10,000 lines.",
    "The file has too much formatting in one paragraph or list. Add blank lines between paragraphs and upload again.",
    "The file nests lists or quotes too deeply.",
    "The file's tables have more than 10,000 cells in total.",
    "The file is too large to show. Make it shorter and upload again.",
    "The file couldn't be read as markdown.",
    "The file name must be 255 characters or fewer.",
  ])("shows the server reason %s", (reason) => {
    expect(getErrorMessage("windows", apiError(reason))).toBe(
      `Couldn’t upload EULA. ${reason}`
    );
  });

  it("maps a wrong file type to the platform's message", () => {
    expect(
      getErrorMessage(
        "windows",
        apiError("The file must be a markdown (.md) file.")
      )
    ).toBe("Couldn’t upload EULA. The file must be a markdown (.md) file.");
    expect(getErrorMessage("darwin", apiError("invalid file type"))).toBe(
      "Couldn’t upload EULA. The file must be a PDF (.pdf)."
    );
  });

  it("falls back to a generic message for anything else", () => {
    expect(getErrorMessage("windows", apiError("database is down"))).toBe(
      "Couldn’t upload EULA. Please try again."
    );
  });
});
