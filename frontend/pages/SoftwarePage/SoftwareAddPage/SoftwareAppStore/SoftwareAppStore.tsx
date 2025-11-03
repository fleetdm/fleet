import React, { useState } from "react";
import { InjectedRouter } from "react-router";
import { SingleValue } from "react-select-5";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import SoftwareAppStoreVpp from "./SoftwareAppStoreVpp";
import SoftwareAppStoreAndroid from "./SoftwareAppStoreAndroid";

const baseClass = "software-app-store";

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const platformOptions = [
  { label: "Apple (macOS, iOS, and iPadOS)", value: "vpp" },
  { label: "Android", value: "android" },
];

const SoftwareAppStore = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
  const [platform, setPlatform] = useState("vpp");

  const onDestinationChange = (
    selectedPlatform: SingleValue<CustomOptionType>
  ) => {
    setPlatform(selectedPlatform?.value || "vpp");
  };

  const renderDropdown = () => (
    <DropdownWrapper
      name="platform"
      label="Platform"
      onChange={onDestinationChange}
      value={platform}
      options={platformOptions}
      className={`${baseClass}__platform-dropdown`}
      wrapperClassname={`${baseClass}__form-field ${baseClass}__form-field--platform`}
      // isDisabled={gitOpsModeEnabled} // TODO: Gitops mode?
    />
  );

  const renderContent = () =>
    platform === "vpp" ? (
      <SoftwareAppStoreVpp currentTeamId={currentTeamId} router={router} />
    ) : (
      <SoftwareAppStoreAndroid currentTeamId={currentTeamId} router={router} />
    );

  return (
    <div className={baseClass}>
      {renderDropdown()}
      {renderContent()}
    </div>
  );
};

export default SoftwareAppStore;
