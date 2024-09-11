import React from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";

const baseClass = "software-package";

interface ISoftwarePackageProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: Location<ISoftwareAddPageQueryParams>;
}

const SoftwarePackage = ({
  currentTeamId,
  router,
  location,
}: ISoftwarePackageProps) => {
  return <div className={baseClass}>Sofware package page</div>;
};

export default SoftwarePackage;
