import React, { useState } from "react";
import { useQuery } from "react-query";

import labelsAPI from "services/entities/labels";
import { ILabelSummary } from "interfaces/label";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

const baseClass = "chart-filter-modal";

const PLATFORM_OPTIONS = [
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
  { label: "ChromeOS", value: "chrome" },
  { label: "iOS", value: "ios" },
  { label: "iPadOS", value: "ipados" },
  { label: "Android", value: "android" },
];

export interface IChartFilterState {
  labelIDs: number[];
  platforms: string[];
}

interface IChartFilterModalProps {
  filters: IChartFilterState;
  onApply: (filters: IChartFilterState) => void;
  onCancel: () => void;
}

const ChartFilterModal = ({
  filters,
  onApply,
  onCancel,
}: IChartFilterModalProps): JSX.Element => {
  const [selectedLabelIDs, setSelectedLabelIDs] = useState<number[]>(
    filters.labelIDs
  );
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(
    filters.platforms
  );

  const { data: labels } = useQuery<ILabelSummary[]>(
    ["labelsSummary"],
    () => labelsAPI.summary().then((res) => res.labels),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 60000,
    }
  );

  const labelOptions = (labels || [])
    .filter((l) => l.label_type !== "builtin")
    .map((l) => ({
      label: l.name,
      value: l.id,
    }));

  const handleApply = () => {
    onApply({
      labelIDs: selectedLabelIDs,
      platforms: selectedPlatforms,
    });
  };

  const handleClear = () => {
    setSelectedLabelIDs([]);
    setSelectedPlatforms([]);
  };

  const hasFilters =
    selectedLabelIDs.length > 0 || selectedPlatforms.length > 0;

  return (
    <Modal title="Chart filters" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__form`}>
        <Dropdown
          label="Labels"
          name="labels"
          options={labelOptions}
          value={selectedLabelIDs.join(",")}
          onChange={(value: string | null) => {
            if (!value) {
              setSelectedLabelIDs([]);
            } else {
              setSelectedLabelIDs(value.split(",").map(Number));
            }
          }}
          multi
          placeholder="All labels"
          searchable
          clearable
        />
        <Dropdown
          label="Platforms"
          name="platforms"
          options={PLATFORM_OPTIONS}
          value={selectedPlatforms.join(",")}
          onChange={(value: string | null) => {
            if (!value) {
              setSelectedPlatforms([]);
            } else {
              setSelectedPlatforms(value.split(","));
            }
          }}
          multi
          placeholder="All platforms"
          searchable={false}
          clearable
        />
      </div>
      <div className={`${baseClass}__btn-wrap`}>
        {hasFilters && (
          <Button variant="text-link" onClick={handleClear}>
            Clear filters
          </Button>
        )}
        <div className={`${baseClass}__btn-actions`}>
          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="default" onClick={handleApply}>
            Apply
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ChartFilterModal;
