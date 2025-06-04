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
  disableOptions?: boolean;
  title: string | null;
}

const TargetChooser = ({
  selectedTarget,
  onSelect,
  disableOptions = false,
  title,
}: ITargetChooserProps) => {
  return (
    <div className="form-field">
      {title && <div className="form-field__label">{title}</div>}
      <Radio
        className={`${baseClass}__radio-input`}
        label="All hosts"
        id="all-hosts-target-radio-btn"
        checked={!disableOptions && selectedTarget === "All hosts"}
        value="All hosts"
        name="target-type"
        onChange={onSelect}
        disabled={disableOptions}
      />
      <Radio
        className={`${baseClass}__radio-input`}
        label="Custom"
        id="custom-target-radio-btn"
        checked={selectedTarget === "Custom"}
        value="Custom"
        name="target-type"
        onChange={onSelect}
        disabled={disableOptions}
      />
    </div>
  );
};

interface ILabelChooserProps {
  isError: boolean;
  isLoading: boolean;
  labels: ILabelSummary[];
  selectedLabels: Record<string, boolean>;
  selectedCustomTarget?: string;
  customTargetOptions?: IDropdownOption[];
  customHelpText?: ReactNode;
  dropdownHelpText?: ReactNode;
  onSelectCustomTarget?: (val: string) => void;
  onSelectLabel: ({ name, value }: { name: string; value: boolean }) => void;
  disableOptions: boolean;
}

const LabelChooser = ({
  isError,
  isLoading,
  labels,
  customHelpText,
  dropdownHelpText,
  selectedLabels,
  selectedCustomTarget,
  customTargetOptions = [],
  onSelectCustomTarget,
  onSelectLabel,
}: ILabelChooserProps) => {
  const getHelpText = (value?: string) => {
    if (dropdownHelpText) return dropdownHelpText;
    return customTargetOptions.find((option) => option.value === value)
      ?.helpText;
  };

  if (isLoading) {
    return <Spinner centered={false} />;
  }

  if (isError) {
    return <DataError />;
  }

  if (!labels.length) {
    return (
      <div className={`${baseClass}__no-labels`}>
        <Link to={PATHS.LABEL_NEW_DYNAMIC}>Add label</Link> to target specific
        hosts.
      </div>
    );
  }

  return (
    <div className={`${baseClass}__custom-label-chooser`}>
      {!!customTargetOptions.length && (
        <Dropdown
          value={selectedCustomTarget}
          options={customTargetOptions}
          searchable={false}
          onChange={onSelectCustomTarget}
        />
      )}
      <div className={`${baseClass}__description`}>
        {customTargetOptions.length
          ? getHelpText(selectedCustomTarget)
          : customHelpText}
      </div>
      <div className={`${baseClass}__checkboxes`}>
        {labels.map((label) => {
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
        })}
      </div>
    </div>
  );
};

interface ITargetLabelSelectorProps {
  selectedTargetType: string;
  selectedCustomTarget?: string;
  customTargetOptions?: IDropdownOption[];
  selectedLabels: Record<string, boolean>;
  labels: ILabelSummary[];
  customHelpText?: ReactNode;
  /** set this prop to show a help text. If it is included then it will override
   * the selected options defined `helpText`
   */
  dropdownHelpText?: ReactNode;
  isLoadingLabels?: boolean;
  isErrorLabels?: boolean;
  className?: string;
  onSelectTargetType: (val: string) => void;
  onSelectCustomTarget?: (val: string) => void;
  onSelectLabel: ({ name, value }: { name: string; value: boolean }) => void;
  disableOptions?: boolean;
  title?: string;
  suppressTitle?: boolean;
}

const TargetLabelSelector = ({
  selectedTargetType,
  selectedCustomTarget,
  customTargetOptions = [],
  selectedLabels,
  dropdownHelpText,
  customHelpText,
  className,
  labels,
  isLoadingLabels = false,
  isErrorLabels = false,
  onSelectTargetType,
  onSelectCustomTarget,
  onSelectLabel,
  disableOptions = false,
  title = "Target",
  suppressTitle = false,
}: ITargetLabelSelectorProps) => {
  const classNames = classnames(baseClass, className, "form");

  return (
    <div className={classNames}>
      <TargetChooser
        selectedTarget={selectedTargetType}
        onSelect={onSelectTargetType}
        disableOptions={disableOptions}
        title={suppressTitle ? null : title}
      />
      {selectedTargetType === "Custom" && (
        <LabelChooser
          selectedCustomTarget={selectedCustomTarget}
          customTargetOptions={customTargetOptions}
          isError={isErrorLabels}
          isLoading={isLoadingLabels}
          labels={labels || []}
          selectedLabels={selectedLabels}
          customHelpText={customHelpText}
          dropdownHelpText={dropdownHelpText}
          onSelectCustomTarget={onSelectCustomTarget}
          onSelectLabel={onSelectLabel}
          disableOptions={disableOptions}
        />
      )}
    </div>
  );
};

export default TargetLabelSelector;
