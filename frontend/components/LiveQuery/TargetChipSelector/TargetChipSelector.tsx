import React from "react";
import {
  ISelectLabel,
  ISelectTeam,
  ISelectTargetsEntity,
} from "interfaces/target";
import Icon from "components/Icon";
import {
  PlatformLabelNameFromAPI,
  LABEL_DISPLAY_MAP,
} from "utilities/constants";

interface ITargetChipSelectorProps {
  entity: ISelectLabel | ISelectTeam;
  isSelected: boolean;
  onClick: (
    value: ISelectLabel | ISelectTeam
  ) => React.MouseEventHandler<HTMLButtonElement>;
}

const isBuiltInLabel = (
  entity: ISelectTargetsEntity
): entity is ISelectLabel & { label_type: "builtin" } => {
  return "label_type" in entity && entity.label_type === "builtin";
};

const TargetChipSelector = ({
  entity,
  isSelected,
  onClick,
}: ITargetChipSelectorProps): JSX.Element => {
  const displayText = (): string => {
    if (isBuiltInLabel(entity)) {
      const labelName = entity.name as PlatformLabelNameFromAPI;
      if (labelName in LABEL_DISPLAY_MAP) {
        return LABEL_DISPLAY_MAP[labelName] || labelName;
      }
    }

    return entity.name || "Missing display name";
  };

  return (
    <button
      className="target-chip-selector"
      data-selected={isSelected}
      onClick={(e) => onClick(entity)(e)}
    >
      <Icon name={isSelected ? "check" : "plus"} />
      <span className="selector-name">{displayText()}</span>
    </button>
  );
};

export default TargetChipSelector;
