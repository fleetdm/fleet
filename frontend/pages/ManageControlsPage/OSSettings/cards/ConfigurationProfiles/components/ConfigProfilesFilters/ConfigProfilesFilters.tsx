import React from "react";
import { useQuery } from "react-query";
import { SingleValue } from "react-select-5";

import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import SearchField from "components/forms/fields/SearchField";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

const baseClass = "config-profiles-filters";

const ALL_LABELS_VALUE = "all";

// Sizes the label dropdown to its longest option at 13px, plus room for the
// filter icon, chevron, and padding, so names don't truncate.
const AVG_CHAR_WIDTH_PX = 7.5;
const DROPDOWN_CHROME_WIDTH_PX = 72;
const MIN_LABEL_DROPDOWN_WIDTH_PX = 150;
const MAX_LABEL_DROPDOWN_WIDTH_PX = 320;

const getLabelDropdownWidth = (labelNames: string[]) => {
  const longest = Math.max(...labelNames.map((name) => name.length));
  const width = longest * AVG_CHAR_WIDTH_PX + DROPDOWN_CHROME_WIDTH_PX;
  return Math.min(
    MAX_LABEL_DROPDOWN_WIDTH_PX,
    Math.max(MIN_LABEL_DROPDOWN_WIDTH_PX, Math.ceil(width))
  );
};

export type ConfigProfilesTypeFilter = "all" | "profile" | "asset";

const TYPE_FILTER_OPTIONS: CustomOptionType[] = [
  { value: "all", label: "All types" },
  { value: "profile", label: "Profiles" },
  { value: "asset", label: "Assets" },
];

interface IConfigProfilesFiltersProps {
  currentTeamId: number;
  showTypeFilter: boolean;
  searchQuery: string;
  typeFilter: ConfigProfilesTypeFilter;
  labelId?: number;
  onSearchQueryChange: (query: string) => void;
  onTypeFilterChange: (type: ConfigProfilesTypeFilter) => void;
  onLabelIdChange: (labelId: number | undefined) => void;
}

const ConfigProfilesFilters = ({
  currentTeamId,
  showTypeFilter,
  searchQuery,
  typeFilter,
  labelId,
  onSearchQueryChange,
  onTypeFilterChange,
  onLabelIdChange,
}: IConfigProfilesFiltersProps) => {
  const { data: labels } = useQuery(
    [{ scope: "labelsSummary", team_id: currentTeamId }],
    () =>
      labelsAPI
        .summary(currentTeamId)
        .then((res) => getCustomLabels(res.labels)),
    { refetchOnWindowFocus: false }
  );

  const labelOptions: CustomOptionType[] = [
    { value: ALL_LABELS_VALUE, label: "All labels" },
    ...(labels ?? []).map((label) => ({
      value: String(label.id),
      label: label.name,
    })),
  ];

  const labelDropdownWidth = getLabelDropdownWidth(
    labelOptions.map((option) => String(option.label))
  );

  const onTypeChange = (option: SingleValue<CustomOptionType>) => {
    if (!option) return;
    onTypeFilterChange(option.value as ConfigProfilesTypeFilter);
  };

  const onLabelChange = (option: SingleValue<CustomOptionType>) => {
    if (!option) return;
    onLabelIdChange(
      option.value === ALL_LABELS_VALUE ? undefined : Number(option.value)
    );
  };

  return (
    <div className={baseClass}>
      {showTypeFilter && (
        <DropdownWrapper
          name="config-profiles-type-filter"
          ariaLabel="Filter by type"
          options={TYPE_FILTER_OPTIONS}
          value={typeFilter}
          onChange={onTypeChange}
          variant="table-filter"
          className={`${baseClass}__type`}
        />
      )}
      <div
        className={`${baseClass}__label`}
        style={{ width: labelDropdownWidth }}
      >
        <DropdownWrapper
          name="config-profiles-label-filter"
          ariaLabel="Filter by label"
          options={labelOptions}
          value={labelId === undefined ? ALL_LABELS_VALUE : String(labelId)}
          onChange={onLabelChange}
          variant="table-filter"
        />
      </div>
      <div className={`${baseClass}__search`}>
        <SearchField
          placeholder="Search by name"
          defaultValue={searchQuery}
          onChange={onSearchQueryChange}
        />
      </div>
    </div>
  );
};

export default ConfigProfilesFilters;
