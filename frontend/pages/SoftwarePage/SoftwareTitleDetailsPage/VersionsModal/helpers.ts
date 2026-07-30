import { compareVersions } from "utilities/helpers";
import { IFleetMaintainedVersion } from "interfaces/software";

/** Radio value meaning "track Latest" (no pin). Sent to the API as an empty
 * `version` field, which the backend treats as clearing the pin. */
export const LATEST_VERSION_VALUE = "";

export interface IVersionOption {
  /** Radio value, also the `version` value PATCHed to the API:
   * "" = Latest, "149.0.7827.54" = exact pin, "^149" = major-version pin. */
  value: string;
  label: string;
}

const majorOf = (version: string): string => version.split(".")[0];

/**
 * Builds the Versions modal radio options from a title's cached versions, in
 * the design's order: "Automatically update to latest", then a "Pin to
 * {version}" per cached version (newest first), then a single "Pin to major
 * version (N)" tracking the latest version's major (stays on N.x, never jumps
 * to N+1).
 */
export const deriveVersionOptions = (
  versions: IFleetMaintainedVersion[]
): IVersionOption[] => {
  const sorted = [...versions].sort((a, b) =>
    compareVersions(b.version, a.version)
  );

  const options: IVersionOption[] = [
    { value: LATEST_VERSION_VALUE, label: "Automatically update to latest" },
  ];

  sorted.forEach((v) => {
    options.push({ value: v.version, label: `Pin to ${v.version}` });
  });

  if (sorted.length) {
    const major = majorOf(sorted[0].version);
    options.push({
      value: `^${major}`,
      label: `Pin to major version (${major})`,
    });
  }

  return options;
};

/** Maps a title's `pinned_version` to the radio value selected when the modal
 * opens: null/undefined/"" → Latest; otherwise the pin string itself
 * ("^149" or an exact version), which matches its option's value. */
export const getPreselectedVersionValue = (
  pinnedVersion: string | null | undefined
): string => pinnedVersion || LATEST_VERSION_VALUE;
