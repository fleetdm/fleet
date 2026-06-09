import { listNamesFromSelectedLabels } from "services/entities/labels";

export type LabelTargetMode = "any" | "all";

export type TargetType = "All hosts" | "Custom";

interface IGenerateCustomTargetLabelKeyArgs {
  targetType: TargetType;
  includeMode: LabelTargetMode;
  includeLabels: Record<string, boolean>;
  excludeLabels: Record<string, boolean>;
  /** Defaults to "any". Profiles only ever exclude "any". */
  excludeMode?: LabelTargetMode;
}

export const generateCustomTargetLabelKey = ({
  targetType,
  includeMode,
  includeLabels,
  excludeLabels,
  excludeMode = "any",
}: IGenerateCustomTargetLabelKeyArgs) => {
  // "All hosts" targets every host, so no label scoping is sent.
  if (targetType !== "Custom") {
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
