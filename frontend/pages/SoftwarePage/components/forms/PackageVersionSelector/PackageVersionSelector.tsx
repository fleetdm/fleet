import React from "react";
import classnames from "classnames";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "package-version-selector";

// This is a temporary solution to disable selecting versions in the UI
// as we currently only support choosing the latest version via gitops.
const disableAllUIOptions = (
  versions: CustomOptionType[],
  selectedVersion: string
): CustomOptionType[] => {
  return versions.map((v: CustomOptionType) => {
    return {
      ...v,
      isDisabled: v.value !== selectedVersion,
    };
  });
};

interface IPackageVersionSelectorProps {
  className?: string;
  versions: CustomOptionType[];
  selectedVersion: string;
  onSelectVersion: (version: string) => void;
}

const PackageVersionSelector = ({
  className,
  versions,
  selectedVersion,
  onSelectVersion,
}: IPackageVersionSelectorProps) => {
  const renderDropdown = () => (
    <DropdownWrapper
      name="package-version-selector"
      className={classnames(baseClass, className)}
      value={selectedVersion as string}
      onChange={(version) => onSelectVersion(version?.value || "")}
      options={disableAllUIOptions(versions, selectedVersion)} // Replace with "versions" when we want to enable selecting versions in the UI
      placeholder="Select a version"
      isDisabled={selectedVersion === versions[0].value}
    />
  );

  return (
    <TooltipWrapper
      tipContent={
        selectedVersion === versions[0].value ? (
          <>
            Currently, you can only use GitOps <br />
            to roll back (UI coming soon).
          </>
        ) : (
          <>
            Currently, to update to latest you have
            <br /> to delete and re-add the software.
          </>
        )
      }
      position="top"
      showArrow
      underline={false}
      tipOffset={8}
    >
      {renderDropdown()}
    </TooltipWrapper>
  );
};

export default PackageVersionSelector;
