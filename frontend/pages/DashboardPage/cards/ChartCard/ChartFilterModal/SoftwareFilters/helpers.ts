// EPSS inputs are entered as a 0–100 percentage; the chart API takes 0.0–1.0.
export const EPSS_MIN_PCT = 0;
export const EPSS_MAX_PCT = 100;

export const EPSS_RANGE_HELP = `Must be from ${EPSS_MIN_PCT} to ${EPSS_MAX_PCT}`;
export const EPSS_RANGE_INVALID_MSG =
  "Minimum EPSS probability cannot be greater than the maximum EPSS probability.";

// At least one software category must stay selected: an empty set is
// indistinguishable from "no filter" on the wire, so the chart would show every
// category instead of none. Block Apply and surface this message instead.
export const NO_CATEGORIES_MSG = "Select at least one software category.";
export const EPSS_RANGE_HELP_MSG = "Enter EPSS values from 0 to 100.";

// Returns an error string when the raw value is out of the 0–100 range, or null
// when it's empty (unset) or valid.
export const getEpssError = (raw: string): string | null => {
  if (raw.trim() === "") {
    return null;
  }
  const n = Number(raw);
  if (Number.isNaN(n) || n < EPSS_MIN_PCT || n > EPSS_MAX_PCT) {
    return EPSS_RANGE_HELP;
  }
  return null;
};

// True when both bounds are present, individually valid, and min > max.
export const isEpssRangeInvalid = (min: string, max: string): boolean => {
  if (min.trim() === "" || max.trim() === "") {
    return false;
  }
  if (getEpssError(min) || getEpssError(max)) {
    return false; // individual range errors are surfaced per-field instead
  }
  return Number(min) > Number(max);
};

// The Software filters are invalid (Apply blocked) when any EPSS field is out of
// range or the min/max are inverted.
export const hasEpssErrors = (min: string, max: string): boolean =>
  getEpssError(min) !== null ||
  getEpssError(max) !== null ||
  isEpssRangeInvalid(min, max);

// An EPSS bound only narrows when min > 0 or max < 100; empty or 0–100 is "all".
export const isEpssActive = (min: string, max: string): boolean => {
  const minActive = min.trim() !== "" && Number(min) > EPSS_MIN_PCT;
  const maxActive = max.trim() !== "" && Number(max) < EPSS_MAX_PCT;
  return minActive || maxActive;
};

// Returns the reason Apply should be blocked for the Software tab, or null when
// the filters are valid. Used both to disable the Apply button and as its
// tooltip text. Categories are checked first since an empty set is the more
// fundamental error.
export const getSoftwareFilterApplyError = (
  categories: string[],
  epssMin: string,
  epssMax: string
): string | null => {
  if (categories.length === 0) {
    return NO_CATEGORIES_MSG;
  }
  if (isEpssRangeInvalid(epssMin, epssMax)) {
    return EPSS_RANGE_INVALID_MSG;
  }
  if (hasEpssErrors(epssMin, epssMax)) {
    return EPSS_RANGE_HELP_MSG;
  }
  return null;
};
