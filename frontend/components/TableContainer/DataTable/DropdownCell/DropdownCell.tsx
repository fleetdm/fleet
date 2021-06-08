import React from "react";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "dropdown-cell";

interface IDropdownCellProps {
  options: IDropdownOption[];
  placeholder: string;
  onChange: (value: any) => void;
}

const DropdownCell = (props: IDropdownCellProps): JSX.Element => {
  const { options, onChange, placeholder } = props;
  return (
    <div className={baseClass}>
      <Dropdown
        onChange={onChange}
        placeholder={placeholder}
        searchable={false}
        options={options}
      />
    </div>
  );
};

export default DropdownCell;
