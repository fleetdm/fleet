export const listNamesFromSelectedLabels = (dict: Record<string, boolean>) => {
  return Object.entries(dict).reduce((acc, [labelName, isSelected]) => {
    if (isSelected) {
      acc.push(labelName);
    }
    return acc;
  }, [] as string[]);
};

export const generateLabelKey = (
  target: string,
  includeMode: "any" | "all",
  includeLabels: Record<string, boolean>,
  excludeMode: "any" | "all",
  excludeLabels: Record<string, boolean>
) => {
  if (target !== "Custom") {
    return {};
  }

  const result: Record<string, string[]> = {};
  const includeNames = listNamesFromSelectedLabels(includeLabels);
  const excludeNames = listNamesFromSelectedLabels(excludeLabels);
  if (includeNames.length) {
    result[
      includeMode === "all" ? "labelsIncludeAll" : "labelsIncludeAny"
    ] = includeNames;
  }
  if (excludeNames.length) {
    result[
      excludeMode === "all" ? "labelsExcludeAll" : "labelsExcludeAny"
    ] = excludeNames;
  }
  return result;
};
