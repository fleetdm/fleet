import React from "react";

import { ISelfServiceCategory } from "interfaces/self_service_category";

import SearchField from "components/forms/fields/SearchField";

import CategoryFilter from "../CategoryFilter";

const baseClass = "software-self-service__header-filters";

interface SelfServiceFiltersProps {
  query: string;
  categoryId?: number;
  categories: ISelfServiceCategory[];
  onSearchQueryChange: (value: string) => void;
  onCategoryChange: (categoryId: number | undefined) => void;
  installAllSlot?: React.ReactNode;
}

const SelfServiceFilters = ({
  query,
  categoryId,
  categories,
  onSearchQueryChange,
  onCategoryChange,
  installAllSlot,
}: SelfServiceFiltersProps) => (
  <div className={baseClass}>
    <CategoryFilter
      categories={categories}
      selectedCategoryId={categoryId}
      onChange={onCategoryChange}
    />
    <div className={`${baseClass}__actions`}>
      {installAllSlot}
      <SearchField
        placeholder="Search by name"
        onChange={onSearchQueryChange}
        defaultValue={query}
      />
    </div>
  </div>
);

export default SelfServiceFilters;
