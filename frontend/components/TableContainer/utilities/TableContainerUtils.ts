const DEFAULT_RESULTS_NAME = "results";

export const generateResultsCountText = (
  name: string = DEFAULT_RESULTS_NAME,
  resultsCount?: number
): string => {
  if (!resultsCount || resultsCount === 0) return `0 ${name}`;
  // If there is 1 result and the last 3 letters in the result
  // name are "ies," we remove the "ies" and add "y"
  // to make the name singular
  if (resultsCount === 1 && name.slice(-3) === "ies") {
    return `${resultsCount} ${name.slice(0, -3)}y`;
  }

  // If there is 1 result and the last 2 letters in the result
  // name are "es," we remove the "es" to make the name singular
  if (resultsCount === 1 && name.slice(-2) === "es") {
    return `${resultsCount} ${name.slice(0, -2)}y`;
  }

  // If there is 1 result and the last letter in the result
  // name is "s," we remove the "s" to make the name singular
  if (resultsCount === 1 && name[name.length - 1] === "s") {
    return `${resultsCount} ${name.slice(0, -1)}`;
  }

  return `${resultsCount.toLocaleString()} ${name}`;
};

export default { generateResultsCountText };
