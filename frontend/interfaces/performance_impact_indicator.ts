export const PerformanceImpactIndicatorValue = {
  MINIMAL: "Minimal",
  CONSIDERABLE: "Considerable",
  EXCESSIVE: "Excessive",
  UNDETERMINED: "Undetermined",
  DENYLISTED: "Denylisted",
} as const;

export type PerformanceImpactIndicator = typeof PerformanceImpactIndicatorValue[keyof typeof PerformanceImpactIndicatorValue];

export const isPerformanceImpactIndicator = (
  value: unknown
): value is PerformanceImpactIndicator => {
  return Object.values(PerformanceImpactIndicatorValue).includes(
    value as PerformanceImpactIndicator
  );
};
