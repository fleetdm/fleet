import { validateSeverityScores } from "components/SeverityFilter";

// EPSS inputs are entered as a 0–100 percentage; the chart API takes 0.0–1.0.
export const EPSS_MIN_PCT = 0;
export const EPSS_MAX_PCT = 100;

// Error copy follows the verb + object + constraint register, with no terminal
// period — see frontend/docs/patterns.md#error-message-copy-register.
export const EPSS_RANGE_HELP = `Enter a probability from ${EPSS_MIN_PCT} to ${EPSS_MAX_PCT}`;
export const EPSS_RANGE_INVALID_MSG =
  "Enter a maximum probability at or above the minimum";

// At least one software category must stay selected: an empty set is
// indistinguishable from "no filter" on the wire, so the chart would show every
// category instead of none.
export const NO_CATEGORIES_MSG = "Select at least one software category";

// Empty is "unset" and never errors.
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

export const isEpssRangeInvalid = (min: string, max: string): boolean => {
  if (min.trim() === "" || max.trim() === "") {
    return false;
  }
  if (getEpssError(min) || getEpssError(max)) {
    return false; // individual range errors are surfaced per-field instead
  }
  return Number(min) > Number(max);
};

// An EPSS bound only narrows when min > 0 or max < 100; empty or 0–100 is "all".
export const isEpssActive = (min: string, max: string): boolean => {
  const minActive = min.trim() !== "" && Number(min) > EPSS_MIN_PCT;
  const maxActive = max.trim() !== "" && Number(max) < EPSS_MAX_PCT;
  return minActive || maxActive;
};

export type SoftwareFilterField =
  | "categories"
  | "epssMin"
  | "epssMax"
  | "cvssMin"
  | "cvssMax";

export type ISoftwareFilterErrors = Partial<
  Record<SoftwareFilterField, string>
>;

export interface ISoftwareFilterFormData {
  categories: string[];
  epssMin: string;
  epssMax: string;
  minScore: string;
  maxScore: string;
}

/**
 * The Software tab's validate function — see
 * frontend/docs/patterns.md#how-to-validate.
 */
export const validateSoftwareFilters = ({
  categories,
  epssMin,
  epssMax,
  minScore,
  maxScore,
}: ISoftwareFilterFormData): ISoftwareFilterErrors => {
  const errors: ISoftwareFilterErrors = {};

  if (categories.length === 0) {
    errors.categories = NO_CATEGORIES_MSG;
  }

  const epssMinError = getEpssError(epssMin);
  const epssMaxError = getEpssError(epssMax);
  if (epssMinError) errors.epssMin = epssMinError;
  if (epssMaxError) errors.epssMax = epssMaxError;
  if (!epssMinError && !epssMaxError && isEpssRangeInvalid(epssMin, epssMax)) {
    errors.epssMax = EPSS_RANGE_INVALID_MSG;
  }

  // Already placed per field; only this tab's names for the two inputs differ.
  const severity = validateSeverityScores({ minScore, maxScore });
  if (severity.minScore) errors.cvssMin = severity.minScore;
  if (severity.maxScore) errors.cvssMax = severity.maxScore;

  return errors;
};
