import React from "react";
import { FilterProps, TableInstance } from "react-table";

import SearchField from "components/forms/fields/SearchField";

const DefaultColumnFilter = ({
  column,
}: FilterProps<TableInstance>): JSX.Element => {
  const { setFilter } = column;

  return (
    <div className={"filter-cell"}>
      <SearchField
        placeholder=""
        onChange={(searchString) => {
          setFilter(searchString || undefined); // Set undefined to remove the filter entirely
        }}
        icon="filter-funnel"
      />
    </div>
  );
};

export default DefaultColumnFilter;
