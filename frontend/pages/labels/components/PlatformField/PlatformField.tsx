import React from "react";
import { noop } from "lodash";

import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import FormField from "components/forms/FormField";

import { LabelPlatform } from "interfaces/label";

// Used to display the platform of an existing label on the edit label page
// (platform is not editable after creation).
const PLATFORM_STRINGS: Record<Exclude<LabelPlatform, "">, string> = {
  darwin: "macOS",
  windows: "Windows",
  linux: "Linux",
  ubuntu: "Ubuntu (Linux)",
  centos: "CentOS (Linux)",
};

interface IPlatformOption extends CustomOptionType {
  value: LabelPlatform;
}

const platformOptions: IPlatformOption[] = [
  { label: "All platforms", value: "" },
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
  { label: "Ubuntu (Linux)", value: "ubuntu" },
  { label: "CentOS (Linux)", value: "centos" },
];

const baseClass = "platform-field";

interface IPlatformFieldProps {
  platform: LabelPlatform;
  isEditing?: boolean;
  onChange?: (platform: LabelPlatform) => void;
}

const PlatformField = ({
  platform,
  isEditing = false,
  onChange = noop,
}: IPlatformFieldProps) => {
  const handleDropdownChange = (newValue: CustomOptionType | null) => {
    // DropdownWrapper passes a SingleValue<CustomOptionType> | null, which
    // widens value to string; the options above only carry LabelPlatform
    // values, so the assertion is safe.
    onChange((newValue?.value ?? "") as LabelPlatform);
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
