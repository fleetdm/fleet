import React from "react";
import { SingleValue } from "react-select-5";
import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import { CATEGORIES_NAV_ITEMS } from "../../helpers";

interface SelfServiceFiltersProps {
  query: string;
  category_id?: number;
  onSearchQueryChange: (value: string) => void;
  onCategoriesDropdownChange: (newValue: SingleValue<CustomOptionType>) => void;
}

const SelfServiceFilters = ({
  query,
  category_id,
  onSearchQueryChange,
  onCategoriesDropdownChange,
}: SelfServiceFiltersProps) => (
  <div className="software-self-service__header-filters">
    <SearchField
      placeholder="Search by name"
      onChange={onSearchQueryChange}
      defaultValue={query}
    />
    <DropdownWrapper
      options={CATEGORIES_NAV_ITEMS.map((category) => ({
        ...category,
        value: String(category.id),
      }))}
      value={String(category_id || 0)}
      onChange={onCategoriesDropdownChange}
      name="categories-dropdown"
      className="software-self-service__categories-dropdown"
    />
  </div>
);

export default SelfServiceFilters;
