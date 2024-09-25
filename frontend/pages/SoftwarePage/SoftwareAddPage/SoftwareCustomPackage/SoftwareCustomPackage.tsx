import React from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";

import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";
import AddSoftwareCustomPackageForm from "./AddSoftwareCustomPackageForm";
import { ICustomPackageAppFormData } from "./AddSoftwareCustomPackageForm/AddSoftwareCustomPackageForm";

const baseClass = "software-custom-package";

interface ISoftwarePackageProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: Location<ISoftwareAddPageQueryParams>;
  isSidePanelOpen: boolean;
  setSidePanelOpen: (isOpen: boolean) => void;
}

const SoftwareCustomPackage = ({
  currentTeamId,
  router,
  location,
  isSidePanelOpen,
  setSidePanelOpen,
}: ISoftwarePackageProps) => {
  const onCancel = () => {
    router.push(
      `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
        team_id: currentTeamId,
      })}`
    );
  };

  const onSubmit = (formData: ICustomPackageAppFormData) => {
    console.log("submit", formData);
  };

  return (
    <div className={baseClass}>
      <AddSoftwareCustomPackageForm
        showSchemaButton={!isSidePanelOpen}
        onClickShowSchema={() => setSidePanelOpen(true)}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
    </div>
  );
};

export default SoftwareCustomPackage;
