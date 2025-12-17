import React from "react";

import {
  ACTIVITY_TYPE_TO_FILTER_LABEL,
  ActivityType,
} from "interfaces/activity";

import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";

import ActivityTypeDropdown from "../ActivityTypeDropdown";

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

// Generate type filter options from ActivityType enum, sort them, and add
// "All types" option at the beginning of the list
const TYPE_FILTER_OPTIONS: { label: string; value: string }[] = Object.values(
  ActivityType
)
  .map((type) => ({
    label: ACTIVITY_TYPE_TO_FILTER_LABEL[type],
    value: type,
  }))
  .sort((a, b) => a.label.localeCompare(b.label));

TYPE_FILTER_OPTIONS.unshift({
  label: "All types",
  value: "",
});

const SORT_OPTIONS = [
  { label: "Sort by newest", value: "desc" },
  { label: "Sort by oldest", value: "asc" },
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
  const onChangeActivityType = (value: string) => {
    setTypeFilter(() => [value]);
    setPageIndex(0);
  };

  return (
    <div className={baseClass}>
      <SearchField
        placeholder="Search activities by user's name or email"
        defaultValue={searchQuery}
        onChange={(value) => {
          setSearchQuery(value);
          setPageIndex(0);
        }}
        icon="search"
      />
      <div className={`${baseClass}__dropdown-filters`}>
        <ActivityTypeDropdown
          className={`${baseClass}__type-filter-dropdown`}
          value={typeFilter[0] || "all"}
          onSelect={onChangeActivityType}
        />
        <DropdownWrapper
          className={`${baseClass}__date-filter-dropdown`}
          iconName="calendar"
          name="date-filter"
          options={DATE_FILTER_OPTIONS}
          value={dateFilter}
          onChange={(value) => {
            if (value === null) return;
            setDateFilter(value.value);
            setPageIndex(0); // Reset to first page on sort change
          }}
        />
        <DropdownWrapper
          className={`${baseClass}__sort-created-at-dropdown`}
          name="created-at-filter"
          iconName="filter"
          options={SORT_OPTIONS}
          value={createdAtDirection}
          onChange={(value) => {
            if (value === null) return;
            if (value.value === createdAtDirection) {
              return; // No change in sort direction
            }
            setCreatedAtDirection(value.value);
            setPageIndex(0); // Reset to first page on sort change
          }}
        />
      </div>
    </div>
  );
};

export default ActivityFeedFilters;
