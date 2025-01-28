import React from "react";
import { FilterProps, TableInstance } from "react-table";

import SearchField from "components/forms/fields/SearchField";

const DefaultColumnFilter = <T extends object>({ column }: FilterProps<T>) => {
  const { setFilter } = column;

  // Remove last_fetched filter per design as it is confusing to filter by a non-displayed date-string
  if (column.id === "last_fetched") {
    return <></>;
  }

  return (
    <div className="filter-cell">
      <SearchField
        placeholder=""
        onClick={(e) => e.stopPropagation()}
        onChange={(searchString) => {
          setFilter(searchString || undefined); // Set undefined to remove the filter entirely
        }}
        icon="filter-funnel"
      />
    </div>
  );
};

export default DefaultColumnFilter;
