import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import {
  ILabelConfig,
  ITargetLabelSelectorProps,
  LabelTargetMode,
  TargetType,
} from "components/TargetLabelSelector";
import labelsAPI, {
  getCustomLabels,
  listNamesFromSelectedLabels,
  ILabelsSummaryResponse,
} from "services/entities/labels";
import { ILabelPolicy, ILabelSummary } from "interfaces/label";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

type LabelSelection = Record<string, boolean>;

const labelsToSelection = (labels: ILabelPolicy[]): LabelSelection =>
  labels.reduce<LabelSelection>((acc, label) => {
    acc[label.name] = true;
    return acc;
  }, {});

interface IBuildPolicyLabelsPayloadArgs {
  targetType: TargetType;
  includeMode: LabelTargetMode;
  includeLabels: LabelSelection;
  excludeMode: LabelTargetMode;
  excludeLabels: LabelSelection;
}

/** Maps the include/exclude selections + modes to the snake_case label arrays
 * the policy API expects. Inactive scopes are sent as empty arrays so the
 * backend clears them. */
const buildPolicyLabelsPayload = ({
  targetType,
  includeMode,
  includeLabels,
  excludeMode,
  excludeLabels,
}: IBuildPolicyLabelsPayloadArgs) => {
  const include =
    targetType === "Custom" ? listNamesFromSelectedLabels(includeLabels) : [];
  const exclude =
    targetType === "Custom" ? listNamesFromSelectedLabels(excludeLabels) : [];
  return {
    labels_include_any: includeMode === "any" ? include : [],
    labels_include_all: includeMode === "all" ? include : [],
    labels_exclude_any: excludeMode === "any" ? exclude : [],
    labels_exclude_all: excludeMode === "all" ? exclude : [],
  };
};

interface IDerivePolicyTargetStateArgs {
  includeAny?: ILabelPolicy[];
  includeAll?: ILabelPolicy[];
  excludeAny?: ILabelPolicy[];
  excludeAll?: ILabelPolicy[];
}

/** Derives the selector's controlled state from a policy's stored label arrays */
const derivePolicyTargetState = ({
  includeAny = [],
  includeAll = [],
  excludeAny = [],
  excludeAll = [],
}: IDerivePolicyTargetStateArgs): {
  targetType: TargetType;
  includeMode: LabelTargetMode;
  includeLabels: LabelSelection;
  excludeMode: LabelTargetMode;
  excludeLabels: LabelSelection;
} => {
  const hasAnyScope = !!(
    includeAny.length ||
    includeAll.length ||
    excludeAny.length ||
    excludeAll.length
  );
  const includeMode: LabelTargetMode = includeAll.length ? "all" : "any";
  const excludeMode: LabelTargetMode = excludeAll.length ? "all" : "any";

  return {
    targetType: hasAnyScope ? "Custom" : "All hosts",
    includeMode,
    includeLabels: labelsToSelection(
      includeMode === "all" ? includeAll : includeAny
    ),
    excludeMode,
    excludeLabels: labelsToSelection(
      excludeMode === "all" ? excludeAll : excludeAny
    ),
  };
};

interface IUsePolicyLabelTargetsArgs {
  includeAny?: ILabelPolicy[];
  includeAll?: ILabelPolicy[];
  excludeAny?: ILabelPolicy[];
  excludeAll?: ILabelPolicy[];
}

export interface IUsePolicyLabelTargets {
  selectorProps: Pick<
    ITargetLabelSelectorProps,
    | "selectedTargetType"
    | "onSelectTargetType"
    | "labels"
    | "isLoadingLabels"
    | "isErrorLabels"
    | "includeConfig"
    | "excludeConfig"
  >;
  selectedTargetType: TargetType;
  hasCustomLabels: boolean;
  getLabelsPayload: () => ReturnType<typeof buildPolicyLabelsPayload>;
}

const usePolicyLabelTargets = ({
  includeAny,
  includeAll,
  excludeAny,
  excludeAll,
}: IUsePolicyLabelTargetsArgs = {}): IUsePolicyLabelTargets => {
  const { isPremiumTier, currentTeam } = useContext(AppContext);

  const [selectedTargetType, setSelectedTargetType] = useState<TargetType>(
    "All hosts"
  );
  const [includeMode, setIncludeMode] = useState<LabelTargetMode>("any");
  const [excludeMode, setExcludeMode] = useState<LabelTargetMode>("any");
  const [includeLabels, setIncludeLabels] = useState<LabelSelection>({});
  const [excludeLabels, setExcludeLabels] = useState<LabelSelection>({});

  const {
    data: labels = [],
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelsSummaryResponse, Error, ILabelSummary[]>(
    ["custom_labels", currentTeam],
    () => labelsAPI.summary(currentTeam?.id, true),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier && !!currentTeam,
      staleTime: 10000,
      select: (res) => getCustomLabels(res.labels),
    }
  );

  // Seed the selection from a policy's stored labels; empty arrays resolve to
  // "All hosts" (create).
  useEffect(() => {
    const seed = derivePolicyTargetState({
      includeAny,
      includeAll,
      excludeAny,
      excludeAll,
    });
    setSelectedTargetType(seed.targetType);
    setIncludeMode(seed.includeMode);
    setIncludeLabels(seed.includeLabels);
    setExcludeMode(seed.excludeMode);
    setExcludeLabels(seed.excludeLabels);
  }, [includeAny, includeAll, excludeAny, excludeAll]);

  const includeConfig: ILabelConfig = {
    selectedLabels: includeLabels,
    onSelectLabel: ({ name, value }) =>
      setIncludeLabels((prev) => ({ ...prev, [name]: value })),
    showModeToggle: true,
    mode: includeMode,
    onSelectMode: setIncludeMode,
    anyTooltip: (
      <>
        Will only target hosts that have{" "}
        <em>
          <b>any</b>
        </em>{" "}
        of these labels.
      </>
    ),
    allTooltip: (
      <>
        Will only target hosts that have{" "}
        <em>
          <b>all</b>
        </em>{" "}
        of these labels.
      </>
    ),
  };

  const excludeConfig: ILabelConfig = {
    selectedLabels: excludeLabels,
    onSelectLabel: ({ name, value }) =>
      setExcludeLabels((prev) => ({ ...prev, [name]: value })),
    showModeToggle: true,
    mode: excludeMode,
    onSelectMode: setExcludeMode,
    anyTooltip: (
      <>
        Will not target hosts that have{" "}
        <em>
          <b>any</b>
        </em>{" "}
        of these labels.
      </>
    ),
    allTooltip: (
      <>
        Will not target hosts that have{" "}
        <em>
          <b>all</b>
        </em>{" "}
        of these labels.
      </>
    ),
  };

  const hasCustomLabels =
    listNamesFromSelectedLabels(includeLabels).length > 0 ||
    listNamesFromSelectedLabels(excludeLabels).length > 0;

  const getLabelsPayload = useCallback(
    () =>
      buildPolicyLabelsPayload({
        targetType: selectedTargetType,
        includeMode,
        includeLabels,
        excludeMode,
        excludeLabels,
      }),
    [selectedTargetType, includeMode, includeLabels, excludeMode, excludeLabels]
  );

  return {
    selectorProps: {
      selectedTargetType,
      onSelectTargetType: setSelectedTargetType,
      labels,
      isLoadingLabels,
      isErrorLabels,
      includeConfig,
      excludeConfig,
    },
    selectedTargetType,
    hasCustomLabels,
    getLabelsPayload,
  };
};

export default usePolicyLabelTargets;
