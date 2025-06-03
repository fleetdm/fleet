import React from "react";

import { ActivityType } from "interfaces/activity";

import SearchField from "components/forms/fields/SearchField";
import ActionsDropdown from "components/ActionsDropdown";

const baseClass = "activity-feed-filters";

const DATE_FILTER_OPTIONS = [
  { label: "All time", value: "all" },
  { label: "Today", value: "today" },
  { label: "Yesterday", value: "yesterday" },
  { label: "Last 7 days", value: "7d" },
  { label: "Last 30 days", value: "30d" },
  { label: "Last 3 months", value: "3m" },
  { label: "Last 12 months", value: "12m" },
];

const TYPE_FILTER_OPTIONS: { label: string; value: string }[] = Object.values(
  ActivityType
)
  .map((type) => ({
    label: type.replace(/_/gi, " ").toLowerCase(),
    value: type,
  }))
  .sort((a, b) => a.label.localeCompare(b.label));

TYPE_FILTER_OPTIONS.unshift({
  label: "All types",
  value: "",
});

const SORT_OPTIONS = [
  { label: "Newest", value: "desc" },
  { label: "Oldest", value: "asc" },
];

interface ActivityFeedFiltersProps {
  searchQuery: string;
  typeFilter: string[];
  dateFilter: string;
  createdAtDirection: string;
  setSearchQuery: (value: string) => void;
  setTypeFilter: (updater: (prev: string[]) => string[]) => void;
  setDateFilter: (value: string) => void;
  setCreatedAtDirection: (value: string) => void;
  setPageIndex: (value: number) => void;
}

const ActivityFeedFilters = ({
  searchQuery,
  setSearchQuery,
  typeFilter,
  setTypeFilter,
  dateFilter,
  setDateFilter,
  createdAtDirection,
  setCreatedAtDirection,
  setPageIndex,
}: ActivityFeedFiltersProps) => {
  const generateTypeFilterLabel = (type: string) => {
    return type ? type.replace(/_/g, " ") : "All types";
  };

  return (
    <div className={baseClass}>
      <SearchField
        placeholder="Search activities by user's name or email..."
        defaultValue={searchQuery}
        onChange={(value) => {
          setSearchQuery(value);
          setPageIndex(0);
        }}
        icon="search"
      />
      <div className={`${baseClass}__dropdown-filters`}>
        <div className={`${baseClass}__filters`}>
          <ActionsDropdown
            className={`${baseClass}__type-filter-dropdown`}
            options={TYPE_FILTER_OPTIONS}
            placeholder={`Type: ${generateTypeFilterLabel(typeFilter[0])}`}
            onChange={(value: string) => {
              setTypeFilter((prev) => {
                // TODO: multiple selections
                return [value];
              });
              setPageIndex(0); // Reset to first page on sort change
            }}
          />
          <ActionsDropdown
            className={`${baseClass}__date-filter-dropdown`}
            options={DATE_FILTER_OPTIONS}
            placeholder={`Date: ${
              DATE_FILTER_OPTIONS.find((option) => option.value === dateFilter)
                ?.label
            }`}
            onChange={(value: string) => {
              setDateFilter(value);
              setPageIndex(0); // Reset to first page on sort change
            }}
          />
        </div>
        <ActionsDropdown
          className={`${baseClass}__sort-created-at-dropdown`}
          options={SORT_OPTIONS}
          placeholder={`Sort by: ${
            createdAtDirection === "asc" ? "Oldest" : "Newest"
          }`}
          onChange={(value: string) => {
            if (value === createdAtDirection) {
              return; // No change in sort direction
            }
            setCreatedAtDirection(value);
            setPageIndex(0); // Reset to first page on sort change
          }}
        />
      </div>
    </div>
  );
};

export default ActivityFeedFilters;
