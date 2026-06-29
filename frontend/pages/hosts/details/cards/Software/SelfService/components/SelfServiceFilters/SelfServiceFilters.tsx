import React from "react";
import classnames from "classnames";

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
}: SelfServiceFiltersProps) => {
  const hasCategories = categories.length > 0;
  return (
    <div
      className={classnames(baseClass, {
        // Drives the narrow-width 2-row layout in SCSS. Without categories the
        // row only holds Install all + Search, which fit on one line at any
        // width, so the wrap rule is skipped.
        [`${baseClass}--with-categories`]: hasCategories,
      })}
    >
      {hasCategories && (
        <CategoryFilter
          categories={categories}
          selectedCategoryId={categoryId}
          onChange={onCategoryChange}
        />
      )}
      {installAllSlot && (
        <div className={`${baseClass}__install-all`}>{installAllSlot}</div>
      )}
      <div className={`${baseClass}__search`}>
        <SearchField
          placeholder="Search by name"
          onChange={onSearchQueryChange}
          defaultValue={query}
        />
      </div>
    </div>
  );
};

export default SelfServiceFilters;
