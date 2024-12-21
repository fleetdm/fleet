import React, { ReactNode } from "react";
import { Link } from "react-router";
import classnames from "classnames";

import PATHS from "router/paths";
import { IDropdownOption } from "interfaces/dropdownOption";
import { ILabelSummary } from "interfaces/label";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Radio from "components/forms/fields/Radio";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "target-label-selector";

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
  customTargetOption: string,
  selectedLabels: Record<string, boolean>
) => {
  if (target !== "Custom") {
    return {};
  }

  return {
    [customTargetOption]: listNamesFromSelectedLabels(selectedLabels),
  };
};

interface ITargetChooserProps {
  selectedTarget: string;
  onSelect: (val: string) => void;
}

const TargetChooser = ({ selectedTarget, onSelect }: ITargetChooserProps) => {
  return (
    <div className={`form-field`}>
      <div className="form-field__label">Target</div>
      <Radio
        className={`${baseClass}__radio-input`}
        label="All hosts"
        id="all-hosts-target-radio-btn"
        checked={selectedTarget === "All hosts"}
        value="All hosts"
        name="target-type"
        onChange={onSelect}
      />
      <Radio
        className={`${baseClass}__radio-input`}
        label="Custom"
        id="custom-target-radio-btn"
        checked={selectedTarget === "Custom"}
        value="Custom"
        name="target-type"
        onChange={onSelect}
      />
    </div>
  );
};

interface ILabelChooserProps {
  isError: boolean;
  isLoading: boolean;
  labels: ILabelSummary[];
  selectedLabels: Record<string, boolean>;
  selectedCustomTarget: string;
  customTargetOptions: IDropdownOption[];
  dropdownHelpText?: ReactNode;
  onSelectCustomTarget: (val: string) => void;
  onSelectLabel: ({ name, value }: { name: string; value: boolean }) => void;
}

const LabelChooser = ({
  isError,
  isLoading,
  labels,
  dropdownHelpText,
  selectedLabels,
  selectedCustomTarget,
  customTargetOptions,
  onSelectCustomTarget,
  onSelectLabel,
}: ILabelChooserProps) => {
  const getHelpText = (value: string) => {
    if (dropdownHelpText) return dropdownHelpText;
    return customTargetOptions.find((option) => option.value === value)
      ?.helpText;
  };

  const renderLabels = () => {
    if (isLoading) {
      return <Spinner centered={false} />;
    }

    if (isError) {
      return <DataError />;
    }

    if (!labels.length) {
      return (
        <div className={`${baseClass}__no-labels`}>
          <span>
            <Link to={PATHS.LABEL_NEW_DYNAMIC}>Add labels</Link> to target
            specific hosts.
          </span>
        </div>
      );
    }

    return labels.map((label) => {
      return (
        <div className={`${baseClass}__label`} key={label.name}>
          <Checkbox
            className={`${baseClass}__checkbox`}
            name={label.name}
            value={!!selectedLabels[label.name]}
            onChange={onSelectLabel}
            parseTarget
          />
          <div className={`${baseClass}__label-name`}>{label.name}</div>
        </div>
      );
    });
  };

  return (
    <div className={`${baseClass}__custom-label-chooser`}>
      <Dropdown
        value={selectedCustomTarget}
        options={customTargetOptions}
        searchable={false}
        onChange={onSelectCustomTarget}
      />
      <div className={`${baseClass}__description`}>
        {getHelpText(selectedCustomTarget)}
      </div>
      <div className={`${baseClass}__checkboxes`}>{renderLabels()}</div>
    </div>
  );
};

interface ITargetLabelSelectorProps {
  selectedTargetType: string;
  selectedCustomTarget: string;
  customTargetOptions: IDropdownOption[];
  selectedLabels: Record<string, boolean>;
  labels: ILabelSummary[];
  /** set this prop to show a help text. If it is encluded then it will override
   * the selected options defined `helpText`
   */
  dropdownHelpText?: ReactNode;
  isLoadingLabels?: boolean;
  isErrorLabels?: boolean;
  className?: string;
  onSelectTargetType: (val: string) => void;
  onSelectCustomTarget: (val: string) => void;
  onSelectLabel: ({ name, value }: { name: string; value: boolean }) => void;
}

const TargetLabelSelector = ({
  selectedTargetType,
  selectedCustomTarget,
  customTargetOptions,
  selectedLabels,
  dropdownHelpText,
  className,
  labels,
  isLoadingLabels = false,
  isErrorLabels = false,
  onSelectTargetType,
  onSelectCustomTarget,
  onSelectLabel,
}: ITargetLabelSelectorProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <TargetChooser
        selectedTarget={selectedTargetType}
        onSelect={onSelectTargetType}
      />
      {selectedTargetType === "Custom" && (
        <LabelChooser
          selectedCustomTarget={selectedCustomTarget}
          customTargetOptions={customTargetOptions}
          isError={isErrorLabels}
          isLoading={isLoadingLabels}
          labels={labels || []}
          selectedLabels={selectedLabels}
          dropdownHelpText={dropdownHelpText}
          onSelectCustomTarget={onSelectCustomTarget}
          onSelectLabel={onSelectLabel}
        />
      )}
    </div>
  );
};

export default TargetLabelSelector;
