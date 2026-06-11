import React, { ReactNode, useState } from "react";
import classnames from "classnames";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { ILabelSummary } from "interfaces/label";
import { listNamesFromSelectedLabels } from "services/entities/labels";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Radio from "components/forms/fields/Radio";
import DataError from "components/DataError";
import Icon from "components/Icon";
import SearchField from "components/forms/fields/SearchField";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";

const baseClass = "target-label-selector";

export type LabelTargetMode = "any" | "all";
export type TargetType = "All hosts" | "Custom";

export interface ILabelConfig {
  selectedLabels: Record<string, boolean>;
  onSelectLabel: (arg: { name: string; value: boolean }) => void;
  /** When true, shows an "Any"/"All" radio that switches this tab between its
   * `_any` and `_all` label scope (e.g. labels_include_any vs
   * labels_include_all). When false, the tab is fixed to its `_any` scope. */
  showModeToggle?: boolean;
  mode?: LabelTargetMode;
  onSelectMode?: (mode: LabelTargetMode) => void;
  anyTooltip?: ReactNode;
  allTooltip?: ReactNode;
}

interface INoLabelsEmptyStateProps {
  description: ReactNode;
  onAddLabel: () => void;
}

const NoLabelsEmptyState = ({
  description,
  onAddLabel,
}: INoLabelsEmptyStateProps) => (
  <div className={`${baseClass}__empty-state`}>
    <span className={`${baseClass}__empty-state--title`}>No labels</span>
    <span className={`${baseClass}__empty-state--description`}>
      {description}
    </span>
    <Button onClick={onAddLabel}>Add label</Button>
  </div>
);

interface ISelectedLabelBadgesProps {
  selectedLabels: Record<string, boolean>;
  onSelectLabel: (arg: { name: string; value: boolean }) => void;
  disableOptions: boolean;
}

const SelectedLabelBadges = ({
  selectedLabels,
  onSelectLabel,
  disableOptions,
}: ISelectedLabelBadgesProps) => {
  const selectedNames = listNamesFromSelectedLabels(selectedLabels);
  if (!selectedNames.length) {
    return null;
  }
  return (
    <div className={`${baseClass}__selected-badges`}>
      {selectedNames.map((name) => (
        <button
          key={name}
          type="button"
          className={`${baseClass}__selected-badge`}
          disabled={disableOptions}
          onClick={() => onSelectLabel({ name, value: false })}
        >
          <span>{name}</span>
          <Icon name="close" size="small" color="ui-fleet-black-75" />
        </button>
      ))}
    </div>
  );
};

interface ILabelCheckboxListProps {
  labels: ILabelSummary[];
  selectedLabels: Record<string, boolean>;
  disabledLabels: Record<string, boolean>;
  onSelectLabel: (arg: { name: string; value: boolean }) => void;
  disableOptions: boolean;
}

const LabelCheckboxList = ({
  labels,
  selectedLabels,
  disabledLabels,
  onSelectLabel,
  disableOptions,
}: ILabelCheckboxListProps) => (
  <div className={`${baseClass}__checkboxes`}>
    {labels.map((label) => (
      <div className={`${baseClass}__label`} key={label.name}>
        <Checkbox
          className={`${baseClass}__checkbox`}
          name={label.name}
          value={!!selectedLabels[label.name]}
          disabled={disableOptions || !!disabledLabels[label.name]}
          onChange={onSelectLabel}
          parseTarget
        >
          {label.name}
        </Checkbox>
      </div>
    ))}
  </div>
);

type LabelTabKey = "include" | "exclude";

interface ILabelModeToggleProps {
  mode: LabelTargetMode;
  onSelectMode?: (mode: LabelTargetMode) => void;
  anyTooltip?: ReactNode;
  allTooltip?: ReactNode;
  tabKey: LabelTabKey;
  disableOptions: boolean;
}

const LabelModeToggle = ({
  mode,
  onSelectMode,
  anyTooltip,
  allTooltip,
  tabKey,
  disableOptions,
}: ILabelModeToggleProps) => {
  // The radio group name is derived from the tab so the two tabs can never
  // share a group.
  const modeName = `${tabKey}-mode`;
  return (
    <div className={`${baseClass}__mode-toggle`}>
      <Radio
        className={`${baseClass}__radio-input`}
        label="Any"
        id={`${modeName}-any-radio`}
        checked={mode === "any"}
        value="any"
        name={modeName}
        tooltip={anyTooltip}
        disabled={disableOptions}
        onChange={(val: string) => onSelectMode?.(val as LabelTargetMode)}
      />
      <Radio
        className={`${baseClass}__radio-input`}
        label="All"
        id={`${modeName}-all-radio`}
        checked={mode === "all"}
        value="all"
        name={modeName}
        tooltip={allTooltip}
        disabled={disableOptions}
        onChange={(val: string) => onSelectMode?.(val as LabelTargetMode)}
      />
    </div>
  );
};

interface ILabelTabContentProps {
  tab: ILabelConfig;
  filteredLabels: ILabelSummary[];
  /** Labels selected in the other tab; disabled here to prevent overlap. */
  disabledLabels: Record<string, boolean>;
  onChangeSearch: (val: string) => void;
  tabKey: LabelTabKey;
  disableOptions: boolean;
}

const LabelTabContent = ({
  tab,
  filteredLabels,
  disabledLabels,
  onChangeSearch,
  tabKey,
  disableOptions,
}: ILabelTabContentProps) => (
  <>
    {tab.showModeToggle && (
      <LabelModeToggle
        mode={tab.mode ?? "any"}
        onSelectMode={tab.onSelectMode}
        anyTooltip={tab.anyTooltip}
        allTooltip={tab.allTooltip}
        tabKey={tabKey}
        disableOptions={disableOptions}
      />
    )}
    <SearchField placeholder="Search labels" onChange={onChangeSearch} />
    <SelectedLabelBadges
      selectedLabels={tab.selectedLabels}
      onSelectLabel={tab.onSelectLabel}
      disableOptions={disableOptions}
    />
    <LabelCheckboxList
      labels={filteredLabels}
      selectedLabels={tab.selectedLabels}
      disabledLabels={disabledLabels}
      onSelectLabel={tab.onSelectLabel}
      disableOptions={disableOptions}
    />
  </>
);

interface ICustomTargetTabsProps {
  labels: ILabelSummary[];
  include: ILabelConfig;
  exclude: ILabelConfig;
  emptyStateDescription: ReactNode;
  onAddLabel: () => void;
  isLoadingLabels: boolean;
  isErrorLabels: boolean;
  disableOptions: boolean;
}

const CustomTargetTabs = ({
  labels,
  include,
  exclude,
  emptyStateDescription,
  onAddLabel,
  isLoadingLabels,
  isErrorLabels,
  disableOptions,
}: ICustomTargetTabsProps) => {
  const [selectedTabIndex, setSelectedTabIndex] = useState(0);
  const [labelSearchQuery, setLabelSearchQuery] = useState("");

  if (isLoadingLabels) {
    return <Spinner centered={false} />;
  }
  if (isErrorLabels) {
    return <DataError />;
  }

  const hasLabels = !!labels.length;
  const filteredLabels = hasLabels
    ? labels.filter((l) =>
        l.name.toLowerCase().includes(labelSearchQuery.toLowerCase())
      )
    : [];

  const onSelectTab = (index: number) => {
    setSelectedTabIndex(index);
    setLabelSearchQuery("");
  };

  return (
    <TabNav secondary>
      <Tabs selectedIndex={selectedTabIndex} onSelect={onSelectTab}>
        <TabList>
          <Tab>
            <TabText
              showCheck={
                listNamesFromSelectedLabels(include.selectedLabels).length > 0
              }
            >
              Include
            </TabText>
          </Tab>
          <Tab>
            <TabText
              showCheck={
                listNamesFromSelectedLabels(exclude.selectedLabels).length > 0
              }
            >
              Exclude
            </TabText>
          </Tab>
        </TabList>
        <TabPanel>
          {hasLabels ? (
            <LabelTabContent
              tab={include}
              filteredLabels={filteredLabels}
              disabledLabels={exclude.selectedLabels}
              onChangeSearch={setLabelSearchQuery}
              tabKey="include"
              disableOptions={disableOptions}
            />
          ) : (
            <NoLabelsEmptyState
              description={emptyStateDescription}
              onAddLabel={onAddLabel}
            />
          )}
        </TabPanel>
        <TabPanel>
          {hasLabels ? (
            <LabelTabContent
              tab={exclude}
              filteredLabels={filteredLabels}
              disabledLabels={include.selectedLabels}
              onChangeSearch={setLabelSearchQuery}
              tabKey="exclude"
              disableOptions={disableOptions}
            />
          ) : (
            <NoLabelsEmptyState
              description={emptyStateDescription}
              onAddLabel={onAddLabel}
            />
          )}
        </TabPanel>
      </Tabs>
    </TabNav>
  );
};

interface ITargetTypeChooserProps {
  selectedTargetType: TargetType;
  onSelectTargetType: (val: TargetType) => void;
  disableOptions?: boolean;
}

const TargetTypeChooser = ({
  selectedTargetType,
  onSelectTargetType,
  disableOptions = false,
}: ITargetTypeChooserProps) => (
  <div className="form-field">
    <Radio
      className={`${baseClass}__radio-input`}
      label="All hosts"
      id="all-hosts-target-radio-btn"
      checked={selectedTargetType === "All hosts"}
      value="All hosts"
      name="target-type"
      onChange={(val: string) => onSelectTargetType(val as TargetType)}
      disabled={disableOptions}
    />
    <Radio
      className={`${baseClass}__radio-input`}
      label="Custom"
      id="custom-target-radio-btn"
      checked={selectedTargetType === "Custom"}
      value="Custom"
      name="target-type"
      onChange={(val: string) => onSelectTargetType(val as TargetType)}
      disabled={disableOptions}
    />
  </div>
);

export interface ITargetLabelSelectorProps {
  selectedTargetType: TargetType;
  onSelectTargetType: (val: TargetType) => void;
  labels: ILabelSummary[];
  includeConfig: ILabelConfig;
  excludeConfig: ILabelConfig;
  emptyStateDescription: ReactNode;
  onAddLabel: () => void;
  isLoadingLabels?: boolean;
  isErrorLabels?: boolean;
  className?: string;
  disableOptions?: boolean;
}

/**
 * TargetLabelSelector lets the user target "All hosts" or a "Custom" set of
 * hosts via a tabbed Include / Exclude experience: one include scope (any/all)
 * may be combined with one exclude scope (any/all) at the same time.
 */
const TargetLabelSelector = ({
  selectedTargetType,
  onSelectTargetType,
  labels,
  includeConfig,
  excludeConfig,
  emptyStateDescription,
  onAddLabel,
  isLoadingLabels = false,
  isErrorLabels = false,
  className,
  disableOptions = false,
}: ITargetLabelSelectorProps) => {
  const classNames = classnames(baseClass, className, "form");

  return (
    <div className={classNames}>
      <TargetTypeChooser
        selectedTargetType={selectedTargetType}
        onSelectTargetType={onSelectTargetType}
        disableOptions={disableOptions}
      />
      {selectedTargetType === "Custom" && (
        <CustomTargetTabs
          labels={labels || []}
          include={includeConfig}
          exclude={excludeConfig}
          emptyStateDescription={emptyStateDescription}
          onAddLabel={onAddLabel}
          isLoadingLabels={isLoadingLabels}
          isErrorLabels={isErrorLabels}
          disableOptions={disableOptions}
        />
      )}
    </div>
  );
};

export default TargetLabelSelector;
