import React from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";

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
    router.push(""); // TODO: path
  };

  const onSubmit = (formData: ICustomPackageAppFormData) => {
    console.log("submit", formData);
  };

  return (
    <div className={baseClass}>
      <AddSoftwareCustomPackageForm
        onClickShowSchemats={() => {}}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
    </div>
  );
};

export default SoftwareCustomPackage;
