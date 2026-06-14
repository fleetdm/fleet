// EPSS inputs are entered as a 0–100 percentage; the chart API takes 0.0–1.0.
export const EPSS_MIN_PCT = 0;
export const EPSS_MAX_PCT = 100;

export const EPSS_RANGE_HELP = `Must be from ${EPSS_MIN_PCT} to ${EPSS_MAX_PCT}`;
export const EPSS_RANGE_INVALID_MSG =
  "Minimum EPSS probability cannot be greater than the maximum EPSS probability.";

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
