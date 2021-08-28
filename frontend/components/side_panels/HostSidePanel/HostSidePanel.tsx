import React, { useState, useCallback } from "react";
import { filter, remove } from "lodash";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import PanelGroup from "components/side_panels/HostSidePanel/PanelGroup";
// @ts-ignore
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";
import { ILabel } from "interfaces/label";
import { OS_CUSTOM_LABELS, OS_DISPLAY_ORDER } from "utilities/constants";

import PlusIcon from "../../../../assets/images/icon-plus-16x16@2x.png";

const baseClass = "host-side-panel";

interface IHostSidePanelProps {
  labels: ILabel[];
  onAddLabelClick: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onLabelClick: (selectedLabel: ILabel) => boolean;
  selectedFilter: string;
  canAddNewLabel: boolean;
}

const HostSidePanel = (props: IHostSidePanelProps): JSX.Element => {
  const {
    labels,
    onAddLabelClick,
    onLabelClick,
    selectedFilter,
    canAddNewLabel,
  } = props;

  const [labelFilter, setLabelFilter] = useState("");

  const onFilterLabels = useCallback(
    (filterString: string): void => {
      setLabelFilter(filterString.toLowerCase());
    },
    [setLabelFilter]
  );

  const allHostLabels = filter(labels, { type: "all" });

  const hostPlatformLabels = (() => {
    const unorderedList: ILabel[] = labels.filter(
      (label) =>
        label.type === "platform" ||
        (label.type === "custom" &&
          label.label_type === "builtin" &&
          OS_CUSTOM_LABELS.includes(label.name))
    );

    let orderedList: ILabel[] = [];
    OS_DISPLAY_ORDER.forEach((name) => {
      orderedList.push(
        ...remove(unorderedList, (label) => label.name === name)
      );
    });
    orderedList = orderedList.concat(unorderedList);

    return orderedList;
  })();

  const customLabels = filter(labels, (label) => {
    const lowerDisplayText = label.display_text.toLowerCase();

    return (
      label.type === "custom" &&
      lowerDisplayText.match(labelFilter) &&
      !OS_CUSTOM_LABELS.includes(label.name)
    );
  });

  return (
    <SecondarySidePanelContainer className={`${baseClass}`}>
      <PanelGroup
        groupItems={allHostLabels}
        onLabelClick={onLabelClick}
        selectedFilter={selectedFilter}
        type="all-hosts"
      />

      <h3>Operating systems</h3>
      <PanelGroup
        groupItems={hostPlatformLabels}
        onLabelClick={onLabelClick}
        selectedFilter={selectedFilter}
        type="platform"
      />
      <div className="title">
        <div>
          <h3>Labels</h3>
        </div>
        <div>
          {canAddNewLabel && (
            <Button
              variant="text-icon"
              onClick={onAddLabelClick}
              className={`${baseClass}__add-label-btn`}
            >
              <span>
                Add label <img src={PlusIcon} alt="Add label icon" />
              </span>
            </Button>
          )}
        </div>
      </div>
      <div
        className={`${baseClass}__panel-group-item ${baseClass}__panel-group-item--filter`}
      >
        <InputField
          name="tags-filter"
          onChange={onFilterLabels}
          placeholder="Filter labels by name..."
          value={labelFilter}
          inputWrapperClass={`${baseClass}__filter-labels`}
        />
      </div>
      <PanelGroup
        groupItems={customLabels}
        onLabelClick={onLabelClick}
        selectedFilter={selectedFilter}
        type="label"
      />
    </SecondarySidePanelContainer>
  );
};

export default HostSidePanel;
