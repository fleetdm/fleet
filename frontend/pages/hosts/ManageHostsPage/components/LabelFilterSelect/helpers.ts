import { ILabel } from "interfaces/label";
import { EMPTY_OPTION, FILTERED_LINUX } from "./constants";

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

export const createDropdownOptions = (labels: ILabel[], labelQuery: string) => {
  const builtInLabels = labels.filter(
    // we filter out All Hosts as that is included in hosts status dropdown filter
    (label) =>
      label.type === "platform" &&
      label.name !== "All Hosts" &&
      !FILTERED_LINUX.includes(label.name)
  );
  const customLabels = labels.filter(
    (label) =>
      label.label_type === "regular" &&
      label.display_text.toLowerCase().includes(labelQuery)
  );
  const customGroupOptions =
    customLabels.length !== 0 ? customLabels : [EMPTY_OPTION];
  const options: IGroupOption[] = [
    createOptionGroup("platform", "Platforms", builtInLabels),
    createOptionGroup("custom", "Labels", customGroupOptions),
  ];
  return options;
};
