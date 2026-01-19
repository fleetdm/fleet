import React from "react";
import { InjectedRouter } from "react-router";
import { SingleValue } from "react-select-5";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import SoftwareAppStoreVpp from "./SoftwareAppStoreVpp";
import SoftwareAppStoreAndroid from "./SoftwareAppStoreAndroid";

const baseClass = "software-app-store";

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      team_id?: string;
      platform?: string;
    };
    search?: string;
  };
}

const platformOptions = [
  { label: "Apple (macOS, iOS, and iPadOS)", value: "apple" },
  { label: "Android", value: "android" },
];

const SoftwareAppStore = ({
  currentTeamId,
  router,
  location,
}: ISoftwareAppStoreProps) => {
  const platform = location.query.platform || "apple";

  const onDestinationChange = (
    selectedPlatform: SingleValue<CustomOptionType>
  ) => {
    router.push(
      getPathWithQueryParams(PATHS.SOFTWARE_ADD_APP_STORE, {
        team_id: currentTeamId,
        platform: selectedPlatform?.value,
      })
    );
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
    />
  );

  const renderContent = () =>
    platform === "apple" ? (
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
