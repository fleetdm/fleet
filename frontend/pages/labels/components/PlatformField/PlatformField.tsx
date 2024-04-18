import React from "react";
import { noop } from "lodash";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import FormField from "components/forms/FormField";

const PLATFORM_STRINGS: { [key: string]: string } = {
  darwin: "macOS",
  windows: "MS Windows",
  ubuntu: "Ubuntu Linux",
  centos: "CentOS Linux",
};

const platformOptions = [
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
  return (
    <div className={baseClass}>
      {!isEditing ? (
        <div className="form-field form-field--dropdown">
          <Dropdown
            label="Platform"
            name="platform"
            onChange={onChange}
            value={platform}
            options={platformOptions}
            classname={`${baseClass}__platform-dropdown`}
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
          />
        </div>
      ) : (
        <FormField
          label="Platform"
          name="platform"
          helpText="Label platforms are immutable. To change the platform, delete this
              label and create a new one."
        >
          <>
            <p>{platform ? PLATFORM_STRINGS[platform] : "All platforms"}</p>
          </>
        </FormField>
      )}
    </div>
  );
};

export default PlatformField;
