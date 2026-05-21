import React from "react";
import { noop } from "lodash";

import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import FormField from "components/forms/FormField";

const PLATFORM_STRINGS: { [key: string]: string } = {
  darwin: "macOS",
  windows: "MS Windows",
  ubuntu: "Ubuntu Linux",
  centos: "CentOS Linux",
};

const platformOptions: CustomOptionType[] = [
  { label: "All platforms", value: "" },
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Ubuntu", value: "ubuntu" },
  { label: "Centos", value: "centos" },
];

const baseClass = "platform-field";

interface IPlatformFieldProps {
  platform: string;
  isEditing?: boolean;
  onChange?: (platform: string) => void;
}

const PlatformField = ({
  platform,
  isEditing = false,
  onChange = noop,
}: IPlatformFieldProps) => {
  const handleDropdownChange = (newValue: CustomOptionType | null) => {
    // DropdownWrapper passes a SingleValue<CustomOptionType> | null
    onChange(newValue?.value ?? "");
  };

  return (
    <div className={baseClass}>
      {!isEditing ? (
        <div className="form-field form-field--dropdown">
          <DropdownWrapper
            label="Platform"
            name="platform"
            onChange={handleDropdownChange}
            // DropdownWrapper accepts either option or string; uses string for simiplicity
            value={platform}
            options={platformOptions}
            className={`${baseClass}__platform-dropdown`}
            wrapperClassname={`${baseClass}__form-field ${baseClass}__form-field--platform`}
            isSearchable={false}
            placeholder="All platforms"
          />
        </div>
      ) : (
        <FormField label="Platform" name="platform">
          <p>{platform ? PLATFORM_STRINGS[platform] : "All platforms"}</p>
        </FormField>
      )}
    </div>
  );
};

export default PlatformField;
