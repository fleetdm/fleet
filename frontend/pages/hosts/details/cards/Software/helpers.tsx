import { QueryParams } from "utilities/url";
import { flatMap } from "lodash";

// available_for_install string > boolean conversion in parseHostSoftwareQueryParams
export const getHostSoftwareFilterFromQueryParams = (
  queryParams: QueryParams
) => {
  const { available_for_install } = queryParams;

  return available_for_install ? "installableSoftware" : "allSoftware";
};

// VERSION COMPARISON UTILITIES FOR SOFTWARE VERSIONS

// Order of pre-release tags for version comparison
const PRE_RELEASE_ORDER = ["alpha", "beta", "rc", ""];

/**
 * Removes build metadata from a version string (e.g., "1.0.0+build" -> "1.0.0").
 */
const stripBuildMetadata = (version: string): string => version.split("+")[0];

/**
 * Splits a version string into an array of numeric and string segments.
 * Handles delimiters, pre-release tags, and normalizes case.
 */
const splitVersion = (version: string): Array<string | number> =>
  flatMap(
    stripBuildMetadata(version).replace(/[-_]/g, ".").split("."),
    (part: string) => part.match(/\d+|[a-zA-Z]+/g) || []
  ).map((seg: string) => (/^\d+$/.test(seg) ? Number(seg) : seg.toLowerCase()));

/**
 * Compares two pre-release identifiers according to PRE_RELEASE_ORDER.
 * Returns -1 if a < b, 1 if a > b, 0 if equal.
 */
const comparePreRelease = (a: string, b: string): number => {
  const idxA = PRE_RELEASE_ORDER.indexOf(a);
  const idxB = PRE_RELEASE_ORDER.indexOf(b);
  if (idxA === -1 && idxB === -1) return a.localeCompare(b);
  if (idxA === -1) return 1;
  if (idxB === -1) return -1;
  if (idxA < idxB) return -1;
  if (idxA > idxB) return 1;
  return 0;
};

/**
 * Compares two software version strings.
 * Returns:
 *   -1 if v1 < v2
 *    0 if v1 === v2
 *    1 if v1 > v2
 * Handles semantic versioning, pre-release tags, and build metadata.
 * See helpers.tests.ts for examples and edge cases.
 */
export const compareVersions = (v1: string, v2: string): number => {
  const s1 = splitVersion(v1);
  const s2 = splitVersion(v2);
  const maxLen = Math.max(s1.length, s2.length);
  let result = 0;

  Array.from({ length: maxLen }).some((_, i) => {
    const a = s1[i] ?? 0;
    const b = s2[i] ?? 0;

    if (typeof a === "number" && typeof b === "number") {
      if (a !== b) {
        result = a > b ? 1 : -1;
        return true;
      }
    } else if (typeof a === "string" && typeof b === "string") {
      // Compare pre-release tags if present
      if (PRE_RELEASE_ORDER.includes(a) || PRE_RELEASE_ORDER.includes(b)) {
        const cmp = comparePreRelease(a, b);
        if (cmp !== 0) {
          result = cmp;
          return true;
        }
      } else if (a !== b) {
        result = a > b ? 1 : -1;
        return true;
      }
    } else {
      // Numbers are always greater than strings (e.g., 1.0 > 1.0-beta)
      result = typeof a === "number" ? 1 : -1;
      return true;
    }
    return false;
  });

  return result;
};
