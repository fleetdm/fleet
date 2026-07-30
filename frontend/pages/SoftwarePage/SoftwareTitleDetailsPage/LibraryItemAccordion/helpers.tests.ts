import { deriveAccordionRowState } from "./helpers";

describe("deriveAccordionRowState", () => {
  it("returns inactive when the row version doesn't match the active version", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "148.0.7778.179",
        activeVersion: "149.0.7827.54",
        pinnedVersion: null,
      })
    ).toEqual({ isActive: false });
  });

  it("returns inactive when activeVersion is null", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "149.0.7827.54",
        activeVersion: null,
        pinnedVersion: null,
      })
    ).toEqual({ isActive: false });
  });

  it("returns inactive when activeVersion is undefined", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "149.0.7827.54",
        activeVersion: undefined,
        pinnedVersion: null,
      })
    ).toEqual({ isActive: false });
  });

  it("returns latest badge for the active row when no pin is set (null)", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "149.0.7827.54",
        activeVersion: "149.0.7827.54",
        pinnedVersion: null,
      })
    ).toEqual({ isActive: true, badgeState: "latest" });
  });

  it("returns latest badge for the active row when no pin is set (undefined)", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "149.0.7827.54",
        activeVersion: "149.0.7827.54",
        pinnedVersion: undefined,
      })
    ).toEqual({ isActive: true, badgeState: "latest" });
  });

  it("returns pinned badge for the active row when pin is an exact version", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "148.0.7778.179",
        activeVersion: "148.0.7778.179",
        pinnedVersion: "148.0.7778.179",
      })
    ).toEqual({ isActive: true, badgeState: "pinned" });
  });

  it("returns majorVersion badge when pin is caret-prefixed", () => {
    expect(
      deriveAccordionRowState({
        rowVersion: "149.0.7827.54",
        activeVersion: "149.0.7827.54",
        pinnedVersion: "^149",
      })
    ).toEqual({ isActive: true, badgeState: "majorVersion" });
  });

  it("does not surface a pin badge on inactive rows even when the pin matches the row version", () => {
    // The pin always applies to the active row, never to an older cached row.
    expect(
      deriveAccordionRowState({
        rowVersion: "148.0.7778.179",
        activeVersion: "149.0.7827.54",
        pinnedVersion: "148.0.7778.179",
      })
    ).toEqual({ isActive: false });
  });
});
