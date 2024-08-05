import { ILabel } from "interfaces/label";
import { getCustomLabels } from "services/entities/labels";

import { EMPTY_OPTION, FILTERED_LINUX, NO_LABELS_OPTION } from "./constants";

export interface IEmptyOption {
  label: string;
  isDisabled: boolean;
}

export interface IGroupOption {
  type: "platform" | "custom";
  label: string;
  options: ILabel[] | IEmptyOption[];
}

const createOptionGroup = (
  type: "platform" | "custom",
  label: string,
  labels: ILabel[] | IEmptyOption[]
) => {
  return {
    type,
    label,
    options: labels,
  };
};

/** Will create the custom label group options and handles when no labels have been created yet or
 * will filter by the desired search query */
const createCustomLabelOptions = (labels: ILabel[], query: string) => {
  const customLabels = getCustomLabels(labels);

  let customLabelGroupOptions: ILabel[] | IEmptyOption[];
  if (customLabels.length === 0) {
    customLabelGroupOptions = [NO_LABELS_OPTION];
  } else {
    const matchingLabels = customLabels.filter((label) =>
      // case-insensitive matching
      label.display_text.toLowerCase().includes(query.toLowerCase())
    );
    customLabelGroupOptions =
      matchingLabels.length !== 0 ? matchingLabels : [EMPTY_OPTION];
  }

  return customLabelGroupOptions;
};

export const createDropdownOptions = (labels: ILabel[], query: string) => {
  const builtInLabels = labels.filter(
    // we filter out All Hosts as that is included in hosts status dropdown filter
    (label) =>
      label.type === "platform" &&
      label.name !== "All Hosts" &&
      !FILTERED_LINUX.includes(label.name)
  );

  const customLabels = createCustomLabelOptions(labels, query);

  const options: IGroupOption[] = [
    createOptionGroup("platform", "Platforms", builtInLabels),
    createOptionGroup("custom", "Labels", customLabels),
  ];
  return options;
};
