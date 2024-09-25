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
}

const SoftwareCustomPackage = ({
  currentTeamId,
  router,
  location,
}: ISoftwarePackageProps) => {
  const onCancel = () => {
    router.push(
      `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
        team_id: location.query.team_id,
      })}`
    );
  };

  const onSubmit = (formData: ICustomPackageAppFormData) => {
    console.log("submit", formData);
  };

  return (
    <div className={baseClass}>
      <AddSoftwareCustomPackageForm
        showSchemaButton
        onClickShowSchema={() => {
          console.log("schema");
        }}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
    </div>
  );
};

export default SoftwareCustomPackage;
