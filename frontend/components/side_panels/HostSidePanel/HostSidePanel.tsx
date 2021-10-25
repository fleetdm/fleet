import React, { useState, useCallback } from "react";
import { filter } from "lodash";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import PanelGroup from "components/side_panels/HostSidePanel/PanelGroup";
// @ts-ignore
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";
import { ILabel } from "interfaces/label";
import { PLATFORM_LABEL_DISPLAY_ORDER } from "utilities/constants";

import PlusIcon from "../../../../assets/images/icon-plus-16x16@2x.png";

const baseClass = "host-side-panel";

interface IHostSidePanelProps {
  labels: ILabel[];
  onAddLabelClick: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onLabelClick: (
    selectedLabel: ILabel
  ) => (evt: React.MouseEvent<HTMLButtonElement>) => void;
  selectedFilter: string | undefined;
  canAddNewLabel: boolean;
}

const HostSidePanel = ({
  labels,
  onAddLabelClick,
  onLabelClick,
  selectedFilter,
  canAddNewLabel,
}: IHostSidePanelProps): JSX.Element => {
  const [labelFilter, setLabelFilter] = useState<string>("");

  const onFilterLabels = useCallback(
    (filterString: string): void => {
      setLabelFilter(filterString.toLowerCase());
    },
    [setLabelFilter]
  );

  const allHostLabels = filter(labels, { type: "all" });

  const hostPlatformLabels = (() => {
    const unorderedList: ILabel[] = labels.filter(
      (label) => label.type === "platform"
    );

    const orderedList: ILabel[] = [];
    PLATFORM_LABEL_DISPLAY_ORDER.forEach((name) => {
      const label = unorderedList.find((el) => el.name === name);
      label && orderedList.push(label);
    });

    return orderedList.filter(
      (label) =>
        ["macOS", "MS Windows", "All Linux"].includes(label.name) ||
        label.count !== 0
    );
  })();

  const customLabels = filter(labels, (label) => {
    const lowerDisplayText = label.display_text.toLowerCase();

    return label.type === "custom" && lowerDisplayText.match(labelFilter);
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
