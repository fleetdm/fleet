import React from "react";
import Select, {
  StylesConfig,
  DropdownIndicatorProps,
  OptionProps,
  components,
} from "react-select-5";

import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "activity-type-dropdown";

interface IActivityTypeDropdownProps {}

const ActivityTypeDropdown = ({}: IActivityTypeDropdownProps) => {
  return <div>test</div>;

  // return (
  //   <div className={baseClass}>
  //     <Select<IDropdownOption, false>
  //       options={options}
  //       placeholder={placeholder}
  //       onChange={handleChange}
  //       isDisabled={disabled}
  //       isSearchable={isSearchable}
  //       styles={customStyles}
  //       components={{
  //         DropdownIndicator: CustomDropdownIndicator,
  //         IndicatorSeparator: () => null,
  //         Option: CustomOption,
  //         SingleValue: () => null, // Doesn't replace placeholder text with selected text
  //         // Note: react-select doesn't support skipping disabled options when keyboarding through
  //       }}
  //       controlShouldRenderValue={false} // Doesn't change placeholder text to selected text
  //       isOptionSelected={() => false} // Hides any styling on selected option
  //       value={null} // Prevent an option from being selected
  //       className={dropdownClassnames}
  //       classNamePrefix={`${baseClass}-select`}
  //       isOptionDisabled={(option) => !!option.disabled}
  //       menuPlacement={menuPlacement}
  //     />
  //   </div>
  // );
};

export default ActivityTypeDropdown;
