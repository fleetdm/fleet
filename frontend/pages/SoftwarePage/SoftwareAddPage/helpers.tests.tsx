import React from "react";
import { render } from "@testing-library/react";

import {
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
  ADD_SOFTWARE_ERROR_PREFIX,
} from "./helpers"; // Adjust path as needed

// --- ensurePeriod tests ---

describe("ensurePeriod", () => {
  it("adds a period to a string that doesn't end with a period", () => {
    expect(ensurePeriod("Test string")).toBe("Test string.");
  });

  it("returns the original string if it already ends with a period", () => {
    expect(ensurePeriod("Test string.")).toBe("Test string.");
  });

  it("returns an empty string unchanged", () => {
    expect(ensurePeriod("")).toBe("");
  });

  it("returns the original string if the string is only a period", () => {
    expect(ensurePeriod(".")).toBe(".");
  });
});

// --- formatAlreadyAvailableInstallMessage tests ---

describe("formatAlreadyAvailableInstallMessage", () => {
  it("returns a React fragment with the correct text and team when the string matches the regex", () => {
    // Example input: "Couldn't add. MyApp already has a package or app available for install on the Marketing team."
    const msg = `${ADD_SOFTWARE_ERROR_PREFIX} MyApp already has a package or app available for install on the Marketing team.`;
    const result = formatAlreadyAvailableInstallMessage(msg);

    // Render for querying text
    const { container } = render(<>{result}</>);

    expect(container.textContent).toContain("Couldn't add.");
    expect(container.textContent).toContain("MyApp");
    expect(container.textContent).toContain("Marketing team");
  });

  it("returns null if the string does not match the expected pattern", () => {
    const msg = "Random error message not matching pattern";
    const result = formatAlreadyAvailableInstallMessage(msg);
    expect(result).toBeNull();
  });

  it("works for different app names and team names", () => {
    const msg = `${ADD_SOFTWARE_ERROR_PREFIX} Zoom already has a package or app available for install on the Engineering team.`;
    const result = formatAlreadyAvailableInstallMessage(msg);

    const { container } = render(<>{result}</>);
    expect(container.textContent).toContain("Zoom");
    expect(container.textContent).toContain("Engineering team");
  });

  it("returns null if the input is empty", () => {
    expect(formatAlreadyAvailableInstallMessage("")).toBeNull();
  });
});
