import { IOperatingSystemVersion } from "interfaces/operating_system";

import {
  compareOSTableVersions,
  compareOSVersionStrings,
} from "./OSTableConfig";

const makeRow = (
  platform: string,
  version: string
): IOperatingSystemVersion => ({
  os_version_id: 1,
  name: `${platform} ${version}`,
  name_only: platform,
  version,
  platform,
  hosts_count: 0,
  vulnerabilities: [],
  kernels: [],
});

// react-table negates a sortType's return value whenever `desc` is true, on
// top of whatever the sortType itself returns. This simulates that so tests
// assert on what actually ends up rendered, not the raw pre-negation value.
const renderedOrder = (
  rowA: IOperatingSystemVersion,
  rowB: IOperatingSystemVersion,
  desc: boolean,
  totals?: Record<string, number>
) => {
  const raw = compareOSTableVersions(rowA, rowB, desc, totals);
  return desc ? -raw : raw;
};

describe("compareOSTableVersions", () => {
  it("falls back to version comparison within the same platform, flipping with direction", () => {
    const a = makeRow("darwin", "26.6");
    const b = makeRow("darwin", "26.10");
    expect(renderedOrder(a, b, false)).toEqual(-1);
    expect(renderedOrder(a, b, true)).toEqual(1);
  });

  it("orders platform groups by host total, most hosts first, regardless of direction", () => {
    // Windows has fewer hosts but a "lower" version number than darwin —
    // grouping should still put windows first because it has more hosts.
    const windowsRow = makeRow("windows", "10.0.9200.100");
    const darwinRow = makeRow("darwin", "26.6");
    const totals = { windows: 3000, darwin: 200 };

    expect(renderedOrder(windowsRow, darwinRow, false, totals)).toBeLessThan(0);
    expect(renderedOrder(darwinRow, windowsRow, false, totals)).toBeGreaterThan(
      0
    );

    // Direction toggle only flips within-group version order, not group
    // order — windows must still sort first when desc is true.
    expect(renderedOrder(windowsRow, darwinRow, true, totals)).toBeLessThan(0);
    expect(renderedOrder(darwinRow, windowsRow, true, totals)).toBeGreaterThan(
      0
    );
  });

  it("falls back to comparing platform names when host totals tie, so groups stay together instead of relying on stable sort", () => {
    const archRow = makeRow("arch", "rolling");
    const debianRow = makeRow("debian", "12");
    // No totals provided at all (both default to 0 — a tie).
    const ascending = renderedOrder(archRow, debianRow, false);
    const descending = renderedOrder(archRow, debianRow, true);

    // Whatever the tiebreak order is, it must be consistent and stay the
    // same regardless of direction — i.e. still deterministic, not a
    // coincidence of input order.
    expect(ascending).not.toEqual(0);
    expect(ascending).toEqual(descending);
    expect(renderedOrder(debianRow, archRow, false)).toEqual(-ascending);
  });
});

describe("compareOSVersionStrings", () => {
  it("compares numeric segments by magnitude, not lexically", () => {
    expect(compareOSVersionStrings("26.6", "26.10")).toEqual(-1);
    expect(compareOSVersionStrings("26.10", "26.6")).toEqual(1);
    expect(compareOSVersionStrings("10.0.9200.100", "10.0.26200.8875")).toEqual(
      -1
    );
  });

  it("compares Windows feature-update codenames by year and half", () => {
    // The reason compareOSVersionStrings exists as its own function instead
    // of reusing the shared frontend/utilities/helpers.tsx compareVersions:
    // that helper doesn't understand this codename shape at all.
    expect(compareOSVersionStrings("21H2", "22H1")).toEqual(-1);
    expect(compareOSVersionStrings("22H1", "21H2")).toEqual(1);
    expect(compareOSVersionStrings("22H1", "22H2")).toEqual(-1);
    expect(compareOSVersionStrings("21H2", "21H2")).toEqual(0);
  });

  it("treats a non-comparable version (e.g. Arch Linux's 'rolling') as older than any comparable version", () => {
    expect(compareOSVersionStrings("rolling", "26.6")).toEqual(-1);
    expect(compareOSVersionStrings("26.6", "rolling")).toEqual(1);
    expect(compareOSVersionStrings("rolling", "rolling")).toEqual(0);
  });

  it("strips Ubuntu's ' LTS' suffix before comparing numerically", () => {
    // osquery's os_version table reports Ubuntu LTS releases with a literal
    // " LTS" suffix (e.g. "22.04.9 LTS"), which Fleet stores verbatim.
    // Without stripping it, these tie as non-comparable instead of
    // comparing numerically.
    expect(compareOSVersionStrings("22.04.9 LTS", "22.04.15 LTS")).toEqual(-1);
    expect(compareOSVersionStrings("22.04.15 LTS", "22.04.9 LTS")).toEqual(1);
    expect(compareOSVersionStrings("22.04.9 lts", "22.04.15 LTS")).toEqual(-1);
  });

  it("treats strings that only coerce to a number via JS's loose Number() as non-comparable, not as valid segments", () => {
    // Number("") === 0, Number("1.") === 1, Number(".1") === 0.1, etc. — a
    // naive `.split(".").map(Number)` would silently accept these as valid
    // version segments, unlike the strict server-side regex this mirrors.
    expect(compareOSVersionStrings("", "26.6")).toEqual(-1);
    expect(compareOSVersionStrings("26.", "26.6")).toEqual(-1);
    expect(compareOSVersionStrings(".26", "26.6")).toEqual(-1);
    expect(compareOSVersionStrings("+1.2", "26.6")).toEqual(-1);
    expect(compareOSVersionStrings("1e2", "26.6")).toEqual(-1);
  });
});
