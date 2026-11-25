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
      <div className={`${baseClass}__dropdowns`}>
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
        <DropdownWrapper
          name="config-profiles-label-filter"
          ariaLabel="Filter by label"
          options={labelOptions}
          value={labelId === undefined ? ALL_LABELS_VALUE : String(labelId)}
          onChange={onLabelChange}
          variant="table-filter"
          isSearchable
          nowrapMenu
          className={`${baseClass}__label`}
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
