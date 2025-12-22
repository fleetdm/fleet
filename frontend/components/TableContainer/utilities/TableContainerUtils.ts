const DEFAULT_RESULTS_NAME = "results";

/**
 * Returns a proper results count text — singular or plural as needed.
 * For regular cases, computes singular by removing "ies" or "s".
 * For irregular cases, pass singularName.
 */
export const generateResultsCountText = (
  name: string = DEFAULT_RESULTS_NAME,
  resultsCount?: number,
  singularName?: string
): string => {
  if (!resultsCount || resultsCount === 0) return `0 ${name}`;

  // If exactly 1 result, return singular form.
  if (resultsCount === 1) {
    if (singularName) {
      return `1 ${singularName}`;
    }
    if (name.endsWith("ies")) {
      return `1 ${name.slice(0, -3)}y`;
    }
    if (name.endsWith("s")) {
      return `1 ${name.slice(0, -1)}`;
    }
  }

  return `${resultsCount.toLocaleString()} ${name}`;
};

export default { generateResultsCountText };
