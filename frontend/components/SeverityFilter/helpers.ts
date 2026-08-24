import { isEmpty } from "lodash";

import numberUtils from "utilities/numbers";
import stringUtils from "utilities/strings/stringUtils";

export const ANY_SEVERITY_VALUE = "any";
export const CUSTOM_SEVERITY_VALUE = "custom";

export type SeverityValue =
  | "any"
  | "critical"
  | "high"
  | "medium"
  | "low"
  | "custom";

export const SEVERITY_HELP_TEXT =
  "CVSS scores (v3) range from 0.0 to 10.0 in 0.1 increments.";

export interface ISeverityOption {
  label: string;
  value: SeverityValue;
  helpText: string;
  /** Bounds the option applies. Undefined on Custom, which reads the inputs. */
  minSeverity?: number;
  maxSeverity?: number;
}

/** The severity selection a parent holds. Scores are raw input strings. */
export interface ISeverityFilterValue {
  severity: SeverityValue;
  minScore: string;
  maxScore: string;
}

const severityName = (severity: SeverityValue) =>
  stringUtils.capitalize(severity);

const severityOptionLabel = (severity: SeverityValue) =>
  `${severityName(severity)} severity`;

const bandHelpText = (min: number, max: number) =>
  `CVSS score ${min.toFixed(1)}-${max}`;

const SEVERITY_BANDS: { value: SeverityValue; min: number; max: number }[] = [
  { value: "critical", min: 9.0, max: 10 },
  { value: "high", min: 7.0, max: 8.9 },
  { value: "medium", min: 4.0, max: 6.9 },
  { value: "low", min: 0.1, max: 3.9 },
];

const CUSTOM_SEVERITY_OPTION: ISeverityOption = {
  label: severityOptionLabel(CUSTOM_SEVERITY_VALUE),
  value: CUSTOM_SEVERITY_VALUE,
  helpText: "Custom CVSS score range",
};

const ANY_SEVERITY_OPTION: ISeverityOption = {
  label: severityOptionLabel(ANY_SEVERITY_VALUE),
  value: ANY_SEVERITY_VALUE,
  helpText: "CVSS score 0-10",
  minSeverity: 0,
  maxSeverity: 10,
};

export const SEVERITY_DROPDOWN_OPTIONS: ISeverityOption[] = [
  ANY_SEVERITY_OPTION,
  ...SEVERITY_BANDS.map(({ value, min, max }) => ({
    label: severityOptionLabel(value),
    value,
    helpText: bandHelpText(min, max),
    minSeverity: min,
    maxSeverity: max,
  })),
  CUSTOM_SEVERITY_OPTION,
];

export const getSeverityOption = (
  severity?: string
): ISeverityOption | undefined =>
  SEVERITY_DROPDOWN_OPTIONS.find((option) => option.value === severity);

export const getSeverityBand = (severity: SeverityValue) =>
  SEVERITY_BANDS.find((band) => band.value === severity);

/** The ends of the CVSS scale. A range spanning both narrows nothing. */
const SCORE_MIN = 0;
const SCORE_MAX = 10;

export const severityForRange = (
  minSeverityValue: number | undefined,
  maxSeverityValue: number | undefined
): SeverityValue => {
  // Loose == so a null from query-param parsing reads as unset too.
  if (minSeverityValue == null && maxSeverityValue == null) {
    return ANY_SEVERITY_VALUE;
  }
  if (minSeverityValue === SCORE_MIN && maxSeverityValue === SCORE_MAX) {
    return ANY_SEVERITY_VALUE;
  }
  const preset = SEVERITY_BANDS.find(
    (band) => band.min === minSeverityValue && band.max === maxSeverityValue
  );
  return preset ? preset.value : CUSTOM_SEVERITY_VALUE;
};

/** The CVSS bounds a selection narrows by. Either both keys, one, or neither. */
export interface ISeverityFilters {
  min?: number;
  max?: number;
}

export const severityFilters = ({
  minScore,
  maxScore,
}: ISeverityScores): ISeverityFilters => {
  const min = minScore === "" ? undefined : Number(minScore);
  const max = maxScore === "" ? undefined : Number(maxScore);

  if (min === SCORE_MIN && max === SCORE_MAX) {
    return {};
  }
  return {
    ...(min !== undefined && { min }),
    ...(max !== undefined && { max }),
  };
};

// How a selection reads where it is summarized away from the control itself.
export const severityValueLabel = ({
  severity,
  minScore,
  maxScore,
}: ISeverityFilterValue): string => {
  const name = severityName(severity);

  if (
    severity === CUSTOM_SEVERITY_VALUE &&
    !isEmpty(severityFilters({ minScore, maxScore }))
  ) {
    const min = minScore === "" ? 0 : Number(minScore);
    const max = maxScore === "" ? 10 : Number(maxScore);
    return `${name} (${min} to ${max})`;
  }

  return name;
};

export const SEVERITY_SCORE_RANGE_ERROR = `Must be from ${SCORE_MIN} to ${SCORE_MAX} in 0.1 increments`;
export const SEVERITY_RANGE_INVALID_MSG = "Must be at or above the minimum";

export type SeverityScoreField = "minScore" | "maxScore";

export type ISeverityScores = Pick<
  ISeverityFilterValue,
  "minScore" | "maxScore"
>;

export type ISeverityFieldErrors = Partial<Record<SeverityScoreField, string>>;

export const validateSeverityScores = ({
  minScore,
  maxScore,
}: ISeverityScores): ISeverityFieldErrors => {
  const isValidScore = (n: number) =>
    numberUtils.isValidNumber(n, SCORE_MIN, SCORE_MAX) &&
    numberUtils.hasAtMostOneDecimal(n);

  const errors: ISeverityFieldErrors = {};
  const min = minScore ? parseFloat(minScore) : undefined;
  const max = maxScore ? parseFloat(maxScore) : undefined;

  if (min !== undefined && !isValidScore(min)) {
    errors.minScore = SEVERITY_SCORE_RANGE_ERROR;
  }
  if (max !== undefined && !isValidScore(max)) {
    errors.maxScore = SEVERITY_SCORE_RANGE_ERROR;
  }
  if (
    min !== undefined &&
    max !== undefined &&
    !errors.minScore &&
    !errors.maxScore &&
    min > max
  ) {
    errors.maxScore = SEVERITY_RANGE_INVALID_MSG;
  }
  return errors;
};
