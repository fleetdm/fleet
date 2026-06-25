/**
 * Value prefixes used in cmdk Command.Item `value` props so the parent
 * dialog's filter function can pass them through unconditionally (the
 * server already filtered the results). Centralized here so a typo in
 * one picker can't silently fall back to cmdk's substring match.
 */
export const RESULT_PREFIXES = {
  host: "HOST_RESULT ",
  software: "SOFTWARE_RESULT ",
  report: "REPORT_RESULT ",
  policy: "POLICY_RESULT ",
} as const;

/**
 * True when a cmdk value should bypass local filtering — used by the
 * dialog's `filter` prop.
 */
export const isPreFilteredResult = (value: string): boolean =>
  Object.values(RESULT_PREFIXES).some((prefix) => value.startsWith(prefix));
